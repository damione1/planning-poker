resource "aws_iam_role_policy" "migration_restore" {
  count = var.migration_in_progress ? 1 : 0
  name  = "TemporaryMigrationRestore"
  role  = aws_iam_role.planning_poker_ssm.id
  policy = jsonencode({
    Version   = "2012-10-17"
    Statement = [{ Effect = "Allow", Action = "s3:GetObject", Resource = "${local.backup_bucket_arn}/planning-poker/*" }]
  })
}
