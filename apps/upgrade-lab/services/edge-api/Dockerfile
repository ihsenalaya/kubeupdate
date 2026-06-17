FROM node:22-alpine
WORKDIR /app
ENV NODE_ENV=production
COPY package*.json ./
RUN npm ci --omit=dev
COPY server.js ./
USER 1000
EXPOSE 3000
CMD ["node", "server.js"]
