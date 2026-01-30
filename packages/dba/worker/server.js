#!/usr/bin/env node
/**
 * DBA Worker Daemon
 *
 * HTTP server that wraps agent-browser commands.
 * Runs on port 39377 (exposed via Morph's worker URL).
 *
 * Authentication:
 * - Validates Stack Auth JWT from Authorization header
 * - Verifies JWT signature using Stack Auth's JWKS
 * - Checks that the user ID in JWT matches the instance owner
 */

const http = require('http');
const https = require('https');
const crypto = require('crypto');
const { spawn } = require('child_process');
const fs = require('fs');
const path = require('path');

const PORT = process.env.PORT || 39377;
const OWNER_ID_FILE = '/var/run/dba/owner-id';
const PROJECT_ID_FILE = '/var/run/dba/stack-project-id';

// Auth configuration - loaded at startup
let ownerId = null;
let projectId = null;
let jwksCache = null;
let jwksCacheTime = 0;
const JWKS_CACHE_TTL = 3600000; // 1 hour

/**
 * Load auth configuration from files
 */
function loadAuthConfig() {
  try {
    ownerId = fs.readFileSync(OWNER_ID_FILE, 'utf8').trim();
    projectId = fs.readFileSync(PROJECT_ID_FILE, 'utf8').trim();
    console.log(`Auth config loaded: owner=${ownerId}, project=${projectId}`);
    return true;
  } catch (e) {
    console.error('Warning: Could not load auth config:', e.message);
    console.error('JWT auth will be disabled. Set up owner-id and stack-project-id files.');
    return false;
  }
}

/**
 * Fetch JWKS from Stack Auth
 */
async function fetchJWKS() {
  // Return cached JWKS if still valid
  if (jwksCache && Date.now() - jwksCacheTime < JWKS_CACHE_TTL) {
    return jwksCache;
  }

  const jwksUrl = `https://api.stack-auth.com/api/v1/projects/${projectId}/.well-known/jwks.json?include_anonymous=true`;

  return new Promise((resolve, reject) => {
    https.get(jwksUrl, (res) => {
      let data = '';
      res.on('data', chunk => data += chunk);
      res.on('end', () => {
        try {
          jwksCache = JSON.parse(data);
          jwksCacheTime = Date.now();
          resolve(jwksCache);
        } catch (e) {
          reject(new Error('Failed to parse JWKS'));
        }
      });
    }).on('error', reject);
  });
}

/**
 * Base64URL decode
 */
function base64UrlDecode(str) {
  str = str.replace(/-/g, '+').replace(/_/g, '/');
  while (str.length % 4) str += '=';
  return Buffer.from(str, 'base64');
}

/**
 * Verify JWT signature using JWKS
 */
async function verifyJWT(token) {
  const parts = token.split('.');
  if (parts.length !== 3) {
    throw new Error('Invalid JWT format');
  }

  const [headerB64, payloadB64, signatureB64] = parts;
  const header = JSON.parse(base64UrlDecode(headerB64).toString());
  const payload = JSON.parse(base64UrlDecode(payloadB64).toString());

  // Check expiration
  if (payload.exp && payload.exp < Date.now() / 1000) {
    throw new Error('JWT expired');
  }

  // Check issuer
  const expectedIssuer = `https://api.stack-auth.com/api/v1/projects/${projectId}`;
  const anonIssuer = `https://api.stack-auth.com/api/v1/projects-anonymous-users/${projectId}`;
  if (payload.iss !== expectedIssuer && payload.iss !== anonIssuer) {
    throw new Error('Invalid issuer');
  }

  // Fetch JWKS and find matching key
  const jwks = await fetchJWKS();
  const key = jwks.keys.find(k => k.kid === header.kid);
  if (!key) {
    throw new Error('Key not found in JWKS');
  }

  // Verify signature using Node.js crypto
  const signatureData = `${headerB64}.${payloadB64}`;
  const signature = base64UrlDecode(signatureB64);

  // Import the JWK as a crypto key
  const keyObject = crypto.createPublicKey({ key, format: 'jwk' });

  const isValid = crypto.verify(
    'sha256',
    Buffer.from(signatureData),
    { key: keyObject, dsaEncoding: 'ieee-p1363' },
    signature
  );

  if (!isValid) {
    throw new Error('Invalid signature');
  }

  return payload;
}

/**
 * Verify authentication - checks JWT and owner ID
 */
async function verifyAuth(req) {
  // If auth config not loaded, deny all requests
  if (!ownerId || !projectId) {
    return { valid: false, error: 'Auth not configured' };
  }

  const authHeader = req.headers['authorization'];
  if (!authHeader) {
    return { valid: false, error: 'No authorization header' };
  }

  const [type, token] = authHeader.split(' ');
  if (type !== 'Bearer' || !token) {
    return { valid: false, error: 'Invalid authorization format' };
  }

  try {
    const payload = await verifyJWT(token);

    // Check if user ID matches owner
    const userId = payload.sub;
    if (userId !== ownerId) {
      return { valid: false, error: 'User is not the instance owner' };
    }

    return { valid: true, userId };
  } catch (e) {
    return { valid: false, error: e.message };
  }
}

// Load auth config at startup
loadAuthConfig();

/**
 * Run an agent-browser command and return the result
 * Uses CDP to connect to the existing Chrome on port 9222
 */
async function runAgentBrowser(args) {
  return new Promise((resolve, reject) => {
    const proc = spawn('agent-browser', [...args, '--cdp', '9222', '--json'], {
      env: { ...process.env, FORCE_COLOR: '0' },
    });

    let stdout = '';
    let stderr = '';

    proc.stdout.on('data', (data) => {
      stdout += data.toString();
    });

    proc.stderr.on('data', (data) => {
      stderr += data.toString();
    });

    proc.on('close', (code) => {
      if (code !== 0) {
        reject(new Error(stderr || stdout || `Exit code ${code}`));
      } else {
        try {
          resolve(JSON.parse(stdout));
        } catch (e) {
          resolve({ success: true, data: stdout });
        }
      }
    });

    proc.on('error', (err) => {
      reject(err);
    });
  });
}

/**
 * Parse JSON body from request
 */
async function parseBody(req) {
  return new Promise((resolve, reject) => {
    let body = '';
    req.on('data', (chunk) => {
      body += chunk.toString();
    });
    req.on('end', () => {
      if (!body) {
        resolve({});
        return;
      }
      try {
        resolve(JSON.parse(body));
      } catch (e) {
        reject(new Error('Invalid JSON'));
      }
    });
    req.on('error', reject);
  });
}

/**
 * Send JSON response
 */
function sendJson(res, data, status = 200) {
  res.writeHead(status, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify(data));
}

/**
 * Handle requests
 */
async function handleRequest(req, res) {
  const url = new URL(req.url, `http://localhost:${PORT}`);
  const reqPath = url.pathname;

  // CORS headers
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET, POST, OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type, Authorization');

  if (req.method === 'OPTIONS') {
    res.writeHead(204);
    res.end();
    return;
  }

  // Health check doesn't require auth
  if (reqPath === '/health') {
    sendJson(res, { status: 'ok' });
    return;
  }

  // All other endpoints require authentication
  const authResult = await verifyAuth(req);
  if (!authResult.valid) {
    sendJson(res, { error: 'Unauthorized', message: authResult.error || 'Authentication required' }, 401);
    return;
  }

  try {
    let result;
    let body = {};

    if (req.method === 'POST') {
      body = await parseBody(req);
    }

    switch (reqPath) {

      case '/snapshot':
        const snapshotArgs = ['snapshot'];
        if (body.interactive) snapshotArgs.push('-i');
        if (body.compact) snapshotArgs.push('-c');
        result = await runAgentBrowser(snapshotArgs);
        break;

      case '/open':
        if (!body.url) {
          sendJson(res, { error: 'url required' }, 400);
          return;
        }
        result = await runAgentBrowser(['open', body.url]);
        break;

      case '/click':
        if (!body.selector) {
          sendJson(res, { error: 'selector required' }, 400);
          return;
        }
        result = await runAgentBrowser(['click', body.selector]);
        break;

      case '/dblclick':
        if (!body.selector) {
          sendJson(res, { error: 'selector required' }, 400);
          return;
        }
        result = await runAgentBrowser(['dblclick', body.selector]);
        break;

      case '/type':
        if (!body.text) {
          sendJson(res, { error: 'text required' }, 400);
          return;
        }
        result = await runAgentBrowser(['type', body.selector || '', body.text]);
        break;

      case '/fill':
        if (!body.selector || body.value === undefined) {
          sendJson(res, { error: 'selector and value required' }, 400);
          return;
        }
        result = await runAgentBrowser(['fill', body.selector, body.value]);
        break;

      case '/press':
        if (!body.key) {
          sendJson(res, { error: 'key required' }, 400);
          return;
        }
        result = await runAgentBrowser(['press', body.key]);
        break;

      case '/hover':
        if (!body.selector) {
          sendJson(res, { error: 'selector required' }, 400);
          return;
        }
        result = await runAgentBrowser(['hover', body.selector]);
        break;

      case '/scroll':
        const dir = body.direction || 'down';
        const amount = body.amount ? String(body.amount) : undefined;
        const scrollArgs = ['scroll', dir];
        if (amount) scrollArgs.push(amount);
        result = await runAgentBrowser(scrollArgs);
        break;

      case '/screenshot':
        // Take screenshot and return base64
        const ssResult = await runAgentBrowser(['screenshot']);
        if (ssResult.success && ssResult.data && ssResult.data.path) {
          const imgData = fs.readFileSync(ssResult.data.path);
          const base64 = imgData.toString('base64');
          result = { success: true, data: { base64 } };
          // Clean up temp file
          try { fs.unlinkSync(ssResult.data.path); } catch (e) {}
        } else {
          result = ssResult;
        }
        break;

      case '/back':
        result = await runAgentBrowser(['back']);
        break;

      case '/forward':
        result = await runAgentBrowser(['forward']);
        break;

      case '/reload':
        result = await runAgentBrowser(['reload']);
        break;

      case '/url':
        result = await runAgentBrowser(['get', 'url']);
        break;

      case '/title':
        result = await runAgentBrowser(['get', 'title']);
        break;

      case '/wait':
        if (!body.selector) {
          sendJson(res, { error: 'selector required' }, 400);
          return;
        }
        result = await runAgentBrowser(['wait', body.selector]);
        break;

      case '/eval':
        if (!body.script) {
          sendJson(res, { error: 'script required' }, 400);
          return;
        }
        result = await runAgentBrowser(['eval', body.script]);
        break;

      default:
        sendJson(res, { error: 'Not found' }, 404);
        return;
    }

    sendJson(res, result);
  } catch (err) {
    console.error('Error:', err.message);
    sendJson(res, { success: false, error: err.message }, 500);
  }
}

const server = http.createServer(handleRequest);

server.listen(PORT, '0.0.0.0', () => {
  console.log(`DBA Worker daemon listening on port ${PORT}`);
});

// Handle graceful shutdown
process.on('SIGTERM', () => {
  console.log('Shutting down...');
  server.close(() => process.exit(0));
});

process.on('SIGINT', () => {
  console.log('Shutting down...');
  server.close(() => process.exit(0));
});
