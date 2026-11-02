resource "aws_iam_role" "github_deploy" {
  name                 = "deploy-planning-poker"
  max_session_duration = 3600
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow", Action = "sts:AssumeRoleWithWebIdentity"
      Principal = { Federated = aws_iam_openid_connect_provider.github.arn }
      Condition = {
        StringEquals = { "token.actions.githubusercontent.com:aud" = "sts.amazonaws.com" }
        StringLike = { "token.actions.githubusercontent.com:sub" = [
          "repo:damione1/planning-poker:ref:refs/tags/v*",
          "repo:damione1/planning-poker:ref:refs/heads/master"
        ] }
      }
    }]
  })
}

resource "aws_iam_role_policy" "github_deploy" {
  name = "DeployOnlyPlanningPoker"
  role = aws_iam_role.github_deploy.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = concat([
      {
        Sid    = "ReadDeploymentResults", Effect = "Allow", Resource = "*"
        Action = ["ssm:GetCommandInvocation"]
      }
      ], var.planning_poker_managed_instance_id == "" ? [] : [
      {
        Sid = "SendApprovedDeploymentDocument", Effect = "Allow", Action = "ssm:SendCommand"
        Resource = [
          "arn:aws:ssm:us-east-1:${local.account_id}:managed-instance/${var.planning_poker_managed_instance_id}",
          aws_ssm_document.planning_poker_deploy.arn,
          aws_ssm_document.planning_poker_health.arn
        ]
      }
    ])
  })
}

resource "aws_iam_role" "planning_poker_ssm" {
  name = "ssm-planning-poker"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow", Principal = { Service = "ssm.amazonaws.com" }, Action = "sts:AssumeRole"
      Condition = {
        StringEquals = { "aws:SourceAccount" = local.account_id }
        ArnLike      = { "aws:SourceArn" = "arn:aws:ssm:us-east-1:${local.account_id}:*" }
      }
    }]
  })
}

resource "aws_iam_role_policy_attachment" "planning_poker_ssm" {
  role       = aws_iam_role.planning_poker_ssm.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

resource "aws_ssm_document" "planning_poker_deploy" {
  name            = "PlanningPoker-Deploy"
  document_type   = "Command"
  document_format = "JSON"
  content = jsonencode({
    schemaVersion = "2.2"
    description   = "Deploy an existing Planning Poker image with the application deployment script"
    parameters = {
      Version  = { type = "String", allowedPattern = "^[0-9]+[.][0-9]+[.][0-9]+(-[A-Za-z0-9.-]+)?$" }
      ImageTag = { type = "String", allowedPattern = "^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$" }
    }
    mainSteps = [{
      action = "aws:runShellScript", name = "Deploy"
      inputs = {
        timeoutSeconds = "600"
        runCommand     = ["set -eu", "sudo -iu ec2-user -- bash /opt/planning-poker/scripts/deploy.sh '{{ Version }}' '{{ ImageTag }}'"]
      }
    }]
  })
}

resource "aws_ssm_document" "planning_poker_health" {
  name            = "PlanningPoker-Health"
  document_type   = "Command"
  document_format = "JSON"
  content = jsonencode({
    schemaVersion = "2.2"
    description   = "Check the registered server identity and the Planning Poker container without changing it"
    mainSteps = [{
      action = "aws:runShellScript", name = "Health"
      inputs = {
        runCommand = ["set -eu", "test \"$(hostname)\" != \"\"", "docker ps --filter name=planning-poker", "curl -fsS --max-time 10 --resolve pokerplanning.net:443:127.0.0.1 https://pokerplanning.net/monitoring/health"]
      }
    }]
  })
}

resource "aws_iam_role_policy" "planning_poker_backups" {
  name = "AppendOnlyPlanningPokerBackups"
  role = aws_iam_role.planning_poker_ssm.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow", Action = "s3:PutObject"
      Resource = "${local.backup_bucket_arn}/planning-poker/*"
    }]
  })
}

resource "aws_ssm_document" "planning_poker_backup" {
  name            = "PlanningPoker-Backup"
  document_type   = "Command"
  document_format = "JSON"
  content = jsonencode({
    schemaVersion = "2.2"
    description   = "Create online SQLite backups, include application files, and append an encrypted S3 archive"
    mainSteps = [{
      action = "aws:runShellScript", name = "Backup"
      inputs = { timeoutSeconds = "900", runCommand = [file("${path.module}/scripts/planning-poker-backup.sh")] }
    }]
  })
}

resource "aws_ssm_association" "planning_poker_backup" {
  count                       = !var.migration_in_progress && var.planning_poker_managed_instance_id != "" ? 1 : 0
  name                        = aws_ssm_document.planning_poker_backup.name
  association_name            = "planning-poker-daily-backup"
  document_version            = "$LATEST"
  schedule_expression         = "cron(0 6 * * ? *)"
  apply_only_at_cron_interval = false
  targets {
    key    = "InstanceIds"
    values = [var.planning_poker_managed_instance_id]
  }
  depends_on = [aws_iam_role_policy.planning_poker_backups]
}
