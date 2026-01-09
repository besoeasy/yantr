# Use Node.js Alpine for smaller image size
FROM node:20-alpine

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

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD node -e "const http=require('http'); const port=process.env.PORT||'5252'; const req=http.get('http://127.0.0.1:'+port+'/api/health', (r)=>{process.exit(r.statusCode===200?0:1)}); req.on('error', ()=>process.exit(1)); req.setTimeout(2000, ()=>{req.destroy(); process.exit(1);});"

# Run the application
CMD ["node", "main.js"]
