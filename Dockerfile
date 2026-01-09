# Use Node.js Alpine for smaller image size
FROM node:24-alpine

# Install Docker CLI
RUN apk add --no-cache docker-cli docker-cli-compose

# Set working directory
WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm install --production

# Copy application files
COPY main.js ./
COPY ui/ ./ui/
COPY apps/ ./apps/

# Expose port
EXPOSE 5252

# Set environment variables
ENV PORT=5252
ENV NODE_ENV=production

# Run the application
CMD ["npm", "start"]
