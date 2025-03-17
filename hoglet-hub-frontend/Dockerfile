FROM node:20-alpine AS base

FROM base AS deps
WORKDIR /app
COPY package.json package-lock.json ./
COPY src/api/generate.mjs src/api/openapi.yaml src/api/
RUN npm ci
RUN mkdir -p /app/src/api/generated && node src/api/generate.mjs

FROM base AS builder
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
COPY --from=deps /app/node_modules ./node_modules
COPY --from=deps /app/src/api/generated ./src/api/generated
COPY . .
RUN npm run build
RUN npm prune --production

FROM base AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=4000

RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs
USER nextjs

COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

EXPOSE 4000
HEALTHCHECK --interval=30s --timeout=30s --start-period=5s --retries=3 \
  CMD node -e "try { require('http').request({port: 4000, path: '/api/health', timeout: 2000}, (res) => process.exit(res.statusCode === 200 ? 0 : 1)).end(); } catch (err) { process.exit(1); }"

CMD ["node", "server.js"]
