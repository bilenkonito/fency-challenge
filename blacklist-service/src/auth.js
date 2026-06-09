// Authentication middleware.
import jwt from 'jsonwebtoken';

/**
 * @param {{ jwtSecret: string, jwtIssuer: string }} config
 * @returns {import('express').RequestHandler}
 */
export function requireAuth(config) {
  return (req, res, next) => {
    const header = req.get('authorization') || '';
    const [scheme, token] = header.split(' ');

    if (scheme !== 'Bearer' || !token) {
      return res.status(401).json({ error: 'missing or malformed Authorization header' });
    }

    try {
      const claims = jwt.verify(token, config.jwtSecret, {
        algorithms: ['HS256'],
        issuer: config.jwtIssuer,
      });
      // Attach a minimal trusted view of the user to the request.
      req.user = { username: claims.username, subject: claims.sub };
      return next();
    } catch {
      // Do not leak the specific verification failure to the client.
      return res.status(401).json({ error: 'invalid or expired token' });
    }
  };
}
