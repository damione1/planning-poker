#!/bin/bash
set -e

echo "🔨 Building Planning Poker for deployment..."

# Create dist directory
mkdir -p dist/package

# Build binary for Linux
echo "→ Building binary..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags="-s -w -X main.Version=manual-$(date +%Y%m%d)" \
  -o dist/package/planning-poker \
  .

chmod +x dist/package/planning-poker

# Copy deployment files
echo "→ Copying deployment files..."
cp deploy/install.sh dist/package/
cp deploy/planning-poker.service dist/package/

# Create README
cat > dist/package/README.txt << 'EOF'
Planning Poker Manual Deployment

Installation:
  sudo ./install.sh

After installation:
  sudo systemctl start planning-poker
  sudo systemctl enable planning-poker
  sudo systemctl status planning-poker

View logs:
  sudo journalctl -u planning-poker -f
EOF

echo "✅ Build complete!"
echo ""
echo "📦 Package contents:"
ls -lh dist/package/
echo ""
echo "🚀 Deploy with:"
echo "  scp -r dist/package/* ubuntu@44.197.23.155:/tmp/planning-poker/"
echo "  ssh ubuntu@44.197.23.155 'cd /tmp/planning-poker && sudo ./install.sh'"
