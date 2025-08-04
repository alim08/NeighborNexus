#!/bin/bash

echo "🚀 Setting up NeighborNexus..."

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker first."
    exit 1
fi

# Check if Docker Compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

# Create .env file if it doesn't exist
if [ ! -f .env ]; then
    echo "📝 Creating .env file..."
    cat > .env << EOF
# Server Configuration
PORT=8080
ENVIRONMENT=development

# Database Configuration
MONGO_URI=mongodb://admin:password@localhost:27017
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# JWT Configuration
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production

# OpenAI Configuration
OPENAI_API_KEY=your-openai-api-key-here

# Pinecone Configuration
PINECONE_API_KEY=your-pinecone-api-key-here
PINECONE_INDEX=neighborenexus
EOF
    echo "✅ .env file created. Please update it with your API keys."
fi

# Start the services
echo "🐳 Starting services with Docker Compose..."
docker-compose up -d

# Wait for services to be ready
echo "⏳ Waiting for services to be ready..."
sleep 10

# Check if services are running
echo "🔍 Checking service status..."
if docker-compose ps | grep -q "Up"; then
    echo "✅ Services are running!"
    echo ""
    echo "🌐 NeighborNexus is now available at:"
    echo "   - Backend API: http://localhost:8080"
    echo "   - Health check: http://localhost:8080/health"
    echo ""
    echo "📝 Next steps:"
    echo "   1. Update the .env file with your OpenAI and Pinecone API keys"
    echo "   2. Restart the backend service: docker-compose restart backend"
    echo "   3. Set up the frontend (optional):"
    echo "      cd frontend"
    echo "      npm install"
    echo "      npm start"
    echo ""
    echo "🔧 To view logs: docker-compose logs -f"
    echo "🛑 To stop services: docker-compose down"
else
    echo "❌ Services failed to start. Check the logs with: docker-compose logs"
    exit 1
fi 