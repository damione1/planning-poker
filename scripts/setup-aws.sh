#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  Planning Poker - AWS Setup Script                        ║${NC}"
echo -e "${BLUE}║  Automated AWS Infrastructure Preparation                  ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Check prerequisites
echo -e "${YELLOW}→ Checking prerequisites...${NC}"

if ! command -v aws &> /dev/null; then
    echo -e "${RED}✗ AWS CLI is not installed${NC}"
    echo "  Install it from: https://aws.amazon.com/cli/"
    exit 1
fi

if ! command -v jq &> /dev/null; then
    echo -e "${YELLOW}⚠ jq is not installed (recommended for better output parsing)${NC}"
    echo "  Install with: brew install jq (macOS) or apt-get install jq (Linux)"
    read -p "Continue anyway? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Check AWS credentials
if ! aws sts get-caller-identity &> /dev/null; then
    echo -e "${RED}✗ AWS credentials are not configured${NC}"
    echo "  Run: aws configure"
    exit 1
fi

echo -e "${GREEN}✓ Prerequisites check passed${NC}"
echo ""

# Get current AWS info
CURRENT_USER=$(aws sts get-caller-identity --query "Arn" --output text 2>/dev/null || echo "unknown")
echo -e "${BLUE}Current AWS Identity:${NC} $CURRENT_USER"
echo ""

# Prompt for AWS region
echo -e "${YELLOW}→ AWS Region Configuration${NC}"
DEFAULT_REGION=$(aws configure get region || echo "us-east-1")
read -p "Enter AWS region [$DEFAULT_REGION]: " AWS_REGION
AWS_REGION=${AWS_REGION:-$DEFAULT_REGION}
echo -e "${GREEN}✓ Using region: $AWS_REGION${NC}"
echo ""

# IAM User Setup
IAM_USER="github-actions-planning-poker"
echo -e "${YELLOW}→ Creating IAM User: $IAM_USER${NC}"

if aws iam get-user --user-name $IAM_USER &> /dev/null; then
    echo -e "${YELLOW}⚠ IAM user already exists${NC}"
    read -p "Do you want to use the existing user? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${RED}✗ Setup cancelled${NC}"
        exit 1
    fi
else
    aws iam create-user --user-name $IAM_USER > /dev/null
    echo -e "${GREEN}✓ IAM user created${NC}"
fi
echo ""

# S3 Bucket Setup
echo -e "${YELLOW}→ Creating S3 Bucket for Terraform State${NC}"
RANDOM_SUFFIX=$(openssl rand -hex 4)
BUCKET_NAME="planning-poker-terraform-state-${RANDOM_SUFFIX}"

echo "  Generated bucket name: $BUCKET_NAME"

# Create bucket with region-specific configuration
if [ "$AWS_REGION" = "us-east-1" ]; then
    aws s3api create-bucket \
        --bucket $BUCKET_NAME \
        --region $AWS_REGION > /dev/null
else
    aws s3api create-bucket \
        --bucket $BUCKET_NAME \
        --region $AWS_REGION \
        --create-bucket-configuration LocationConstraint=$AWS_REGION > /dev/null
fi

echo -e "${GREEN}✓ S3 bucket created${NC}"

# Enable versioning
aws s3api put-bucket-versioning \
    --bucket $BUCKET_NAME \
    --versioning-configuration Status=Enabled > /dev/null
echo -e "${GREEN}✓ Versioning enabled${NC}"

# Enable encryption
aws s3api put-bucket-encryption \
    --bucket $BUCKET_NAME \
    --server-side-encryption-configuration '{
      "Rules": [{
        "ApplyServerSideEncryptionByDefault": {
          "SSEAlgorithm": "AES256"
        }
      }]
    }' > /dev/null
echo -e "${GREEN}✓ Encryption enabled${NC}"

# Block public access
aws s3api put-public-access-block \
    --bucket $BUCKET_NAME \
    --public-access-block-configuration \
        BlockPublicAcls=true,\
IgnorePublicAcls=true,\
BlockPublicPolicy=true,\
RestrictPublicBuckets=true > /dev/null
echo -e "${GREEN}✓ Public access blocked${NC}"
echo ""

# Update IAM Policy
echo -e "${YELLOW}→ Configuring IAM Policy${NC}"
POLICY_FILE="$PROJECT_ROOT/terraform/iam-policy.json"

if [ ! -f "$POLICY_FILE" ]; then
    echo -e "${RED}✗ Policy file not found: $POLICY_FILE${NC}"
    exit 1
fi

# Create backup
cp "$POLICY_FILE" "$POLICY_FILE.backup"

# Update bucket name in policy
sed -i.tmp "s|planning-poker-terraform-state-YOUR-UNIQUE-ID|$BUCKET_NAME|g" "$POLICY_FILE"
rm -f "$POLICY_FILE.tmp"
echo -e "${GREEN}✓ IAM policy updated with bucket name${NC}"

# Create IAM policy
POLICY_NAME="GitHubActionsLightsailPolicy"
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

# Check if policy exists
POLICY_ARN="arn:aws:iam::${AWS_ACCOUNT_ID}:policy/${POLICY_NAME}"
if aws iam get-policy --policy-arn $POLICY_ARN &> /dev/null; then
    echo -e "${YELLOW}⚠ IAM policy already exists${NC}"
    read -p "Do you want to create a new version? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        # Get current version count and delete oldest if at limit
        VERSIONS=$(aws iam list-policy-versions --policy-arn $POLICY_ARN --query 'Versions[?IsDefaultVersion==`false`].VersionId' --output text)
        VERSION_COUNT=$(echo $VERSIONS | wc -w)

        if [ $VERSION_COUNT -ge 4 ]; then
            OLDEST_VERSION=$(echo $VERSIONS | awk '{print $1}')
            aws iam delete-policy-version --policy-arn $POLICY_ARN --version-id $OLDEST_VERSION
            echo -e "${GREEN}✓ Deleted oldest policy version${NC}"
        fi

        aws iam create-policy-version \
            --policy-arn $POLICY_ARN \
            --policy-document file://$POLICY_FILE \
            --set-as-default > /dev/null
        echo -e "${GREEN}✓ IAM policy updated${NC}"
    fi
else
    aws iam create-policy \
        --policy-name $POLICY_NAME \
        --policy-document file://$POLICY_FILE \
        --description "GitHub Actions permissions for Planning Poker deployment" > /dev/null
    echo -e "${GREEN}✓ IAM policy created${NC}"
fi

# Attach policy to user
if aws iam list-attached-user-policies --user-name $IAM_USER | grep -q $POLICY_NAME; then
    echo -e "${YELLOW}⚠ Policy already attached to user${NC}"
else
    aws iam attach-user-policy \
        --user-name $IAM_USER \
        --policy-arn $POLICY_ARN
    echo -e "${GREEN}✓ Policy attached to user${NC}"
fi
echo ""

# Create Access Keys
echo -e "${YELLOW}→ Creating Access Keys${NC}"
read -p "Do you want to create new access keys? (y/n) " -n 1 -r
echo

if [[ $REPLY =~ ^[Yy]$ ]]; then
    # Check existing keys
    EXISTING_KEYS=$(aws iam list-access-keys --user-name $IAM_USER --query 'AccessKeyMetadata[*].AccessKeyId' --output text)
    KEY_COUNT=$(echo $EXISTING_KEYS | wc -w)

    if [ $KEY_COUNT -ge 2 ]; then
        echo -e "${RED}✗ User already has 2 access keys (AWS limit)${NC}"
        echo "  Existing keys: $EXISTING_KEYS"
        echo "  You must delete one before creating a new key"
        ACCESS_KEY_ID="<use-existing-key>"
        SECRET_ACCESS_KEY="<use-existing-key>"
    else
        KEYS_OUTPUT=$(aws iam create-access-key --user-name $IAM_USER)
        ACCESS_KEY_ID=$(echo $KEYS_OUTPUT | grep -o '"AccessKeyId": "[^"]*' | cut -d'"' -f4)
        SECRET_ACCESS_KEY=$(echo $KEYS_OUTPUT | grep -o '"SecretAccessKey": "[^"]*' | cut -d'"' -f4)
        echo -e "${GREEN}✓ Access keys created${NC}"
    fi
else
    ACCESS_KEY_ID="<use-existing-key>"
    SECRET_ACCESS_KEY="<use-existing-key>"
fi
echo ""

# Summary
echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  Setup Complete! 🎉                                        ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${GREEN}AWS Resources Created:${NC}"
echo "  • IAM User:      $IAM_USER"
echo "  • IAM Policy:    $POLICY_NAME"
echo "  • S3 Bucket:     $BUCKET_NAME"
echo "  • AWS Region:    $AWS_REGION"
echo ""

echo -e "${YELLOW}════════════════════════════════════════════════════════════${NC}"
echo -e "${YELLOW}GitHub Secrets Configuration${NC}"
echo -e "${YELLOW}════════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${BLUE}Run these commands to configure GitHub secrets:${NC}"
echo ""
echo -e "${GREEN}# Set AWS credentials${NC}"
echo "gh secret set AWS_ACCESS_KEY_ID --body \"$ACCESS_KEY_ID\""
echo "gh secret set AWS_SECRET_ACCESS_KEY --body \"$SECRET_ACCESS_KEY\""
echo ""
echo -e "${GREEN}# Set Terraform state bucket${NC}"
echo "gh secret set TF_STATE_BUCKET --body \"$BUCKET_NAME\""
echo ""
echo -e "${GREEN}# Set AWS region${NC}"
echo "gh variable set AWS_REGION --body \"$AWS_REGION\""
echo ""

# Save to file
OUTPUT_FILE="$PROJECT_ROOT/aws-setup-output.txt"
cat > "$OUTPUT_FILE" << EOF
Planning Poker - AWS Setup Output
Generated: $(date)

IAM User: $IAM_USER
IAM Policy: $POLICY_NAME
S3 Bucket: $BUCKET_NAME
AWS Region: $AWS_REGION
AWS Account ID: $AWS_ACCOUNT_ID

Access Key ID: $ACCESS_KEY_ID
Secret Access Key: $SECRET_ACCESS_KEY

GitHub CLI Commands:
--------------------
gh secret set AWS_ACCESS_KEY_ID --body "$ACCESS_KEY_ID"
gh secret set AWS_SECRET_ACCESS_KEY --body "$SECRET_ACCESS_KEY"
gh secret set TF_STATE_BUCKET --body "$BUCKET_NAME"
gh variable set AWS_REGION --body "$AWS_REGION"

Policy ARN: $POLICY_ARN
EOF

echo -e "${YELLOW}════════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${GREEN}✓ Configuration saved to: aws-setup-output.txt${NC}"
echo -e "${YELLOW}⚠ Keep this file secure - it contains sensitive credentials!${NC}"
echo ""

echo -e "${BLUE}Next Steps:${NC}"
echo "  1. Run the GitHub CLI commands above to configure secrets"
echo "  2. Update terraform/production.tfvars with your settings"
echo "  3. Test the CI/CD pipeline by creating a PR"
echo ""
echo -e "${GREEN}Setup complete! You're ready to deploy.${NC}"
