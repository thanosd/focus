# Build stage — pinned to the build host's native arch: the compiled output
# (.next/standalone) is plain JavaScript, so one native build serves both
# halves of a multi-arch image instead of repeating npm ci + next build
# under QEMU (which took the arm64 half from minutes to ~an hour in CI).
FROM --platform=$BUILDPLATFORM node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install all dependencies (including devDependencies needed for build)
RUN npm ci

# next bundles sharp (native image optimization) into the standalone
# output. This stage cross-builds for both architectures, so install BOTH
# musl arch variants — sharp's loader picks the matching one at runtime.
# hadolint ignore=DL3016
RUN npm install --force --no-save --os=linux --libc=musl --cpu=x64 sharp \
    && npm install --force --no-save --os=linux --libc=musl --cpu=arm64 sharp

# Copy source code
COPY . .

# Build argument for API URL
ARG NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_API_URL=${NEXT_PUBLIC_API_URL}


# Build the application
RUN npm run build

# The runner stage may be a different architecture than this build stage.
# Verify the standalone output is arch-complete: both musl sharp variants
# must be present, and no native modules other than sharp's may appear
# (anything else would be single-arch and break one platform).
# hadolint ignore=DL4006
RUN find .next/standalone -name '*.node' > /tmp/native-modules.txt && \
    cat /tmp/native-modules.txt && \
    grep -q sharp-linuxmusl-x64 /tmp/native-modules.txt && \
    grep -q sharp-linuxmusl-arm64 /tmp/native-modules.txt && \
    ! grep -v '@img/sharp-' /tmp/native-modules.txt | grep -q .

# Production stage
FROM node:20-alpine AS runner

WORKDIR /app

# Set to production
ENV NODE_ENV=production

# Create non-root user
RUN addgroup --system --gid 1001 nodejs && \
    adduser --system --uid 1001 nextjs

# Copy necessary files from builder
COPY --from=builder --chown=nextjs:nodejs /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

# Switch to non-root user
USER nextjs

# Expose port
EXPOSE 3000

# Set hostname
ENV HOSTNAME="0.0.0.0"

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3000 || exit 1

# Run the application
CMD ["node", "server.js"]