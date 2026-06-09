FROM node:22-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
# Only production dependencies in the final image.
RUN npm ci --omit=dev

FROM node:22-alpine AS runtime
WORKDIR /app
ENV NODE_ENV=production

COPY --from=deps /app/node_modules ./node_modules
COPY package.json ./
COPY src ./src

# Run as the unprivileged built-in "node" user.
USER node
EXPOSE 3000
CMD ["node", "src/server.js"]
