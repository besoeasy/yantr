import { spawn } from "child_process";

/**
 * Helper function to spawn a process and capture output
 * @param {string} command - The command to execute
 * @param {string[]} args - Array of arguments to pass to the command
 * @param {Object} options - Options to pass to spawn (cwd, env, etc.)
 * @returns {Promise<{stdout: string, stderr: string, exitCode: number}>}
 */
export function spawnProcess(command, args, options = {}) {
  return new Promise((resolve, reject) => {
    const proc = spawn(command, args, {
      ...options,
      env: options.env || process.env,
    });

    let stdout = '';
    let stderr = '';

    if (proc.stdout) {
      proc.stdout.on('data', (data) => {
        stdout += data.toString();
      });
    }

    if (proc.stderr) {
      proc.stderr.on('data', (data) => {
        stderr += data.toString();
      });
    }

    proc.on('close', (code) => {
      resolve({ stdout, stderr, exitCode: code });
    });

    proc.on('error', (err) => {
      reject(err);
    });
  });
}

// ============================================================================
// ERROR HANDLING
// ============================================================================

/**
 * Base application error class
 */
export class AppError extends Error {
  constructor(message, statusCode = 500, details = null) {
    super(message);
    this.name = this.constructor.name;
    this.statusCode = statusCode;
    this.details = details;
    this.isOperational = true; // Distinguish operational errors from programming errors
    Error.captureStackTrace(this, this.constructor);
  }
}

/**
 * Bad request error (400)
 */
export class BadRequestError extends AppError {
  constructor(message, details = null) {
    super(message, 400, details);
  }
}

/**
 * Not found error (404)
 */
export class NotFoundError extends AppError {
  constructor(message, details = null) {
    super(message, 404, details);
  }
}

/**
 * Conflict error (409)
 */
export class ConflictError extends AppError {
  constructor(message, details = null) {
    super(message, 409, details);
  }
}

/**
 * Docker-specific error
 */
export class DockerError extends AppError {
  constructor(message, details = null) {
    super(message, 500, details);
  }
}

/**
 * Express error handling middleware
 * Should be the last middleware added to the app
 * @param {Error} err - The error object
 * @param {import('express').Request} req - Express request
 * @param {import('express').Response} res - Express response
 * @param {import('express').NextFunction} next - Express next function
 */
export function errorHandler(err, req, res, next) {
  // Log error for debugging
  const timestamp = new Date().toISOString();
  console.error(`[${timestamp}] ❌ [ERROR] ${req.method} ${req.path}`);
  console.error(`[${timestamp}] ❌ [ERROR] ${err.stack || err.message}`);

  // Don't send error response if headers already sent
  if (res.headersSent) {
    return next(err);
  }

  // Handle operational errors (errors we expect and handle)
  if (err.isOperational) {
    return res.status(err.statusCode).json({
      success: false,
      error: err.message,
      details: err.details,
    });
  }

  // Handle Docker API errors
  if (err.statusCode && err.reason) {
    return res.status(err.statusCode >= 400 && err.statusCode < 600 ? err.statusCode : 500).json({
      success: false,
      error: err.reason || err.message,
      details: err.json || null,
    });
  }

  // Handle unexpected errors (programming errors)
  // Don't leak error details in production
  const isDevelopment = process.env.NODE_ENV === 'development';
  
  return res.status(500).json({
    success: false,
    error: "Internal server error",
    details: isDevelopment ? {
      message: err.message,
      stack: err.stack,
    } : null,
  });
}

/**
 * Async handler wrapper to catch errors in async route handlers
 * Eliminates the need for try-catch in every route
 * @param {Function} fn - Async route handler function
 * @returns {Function} Wrapped function
 */
export function asyncHandler(fn) {
  return (req, res, next) => {
    Promise.resolve(fn(req, res, next)).catch(next);
  };
}

/**
 * Extract base app ID from compose project name
 * Removes instance suffixes like -2, -3, etc.
 * @param {string} composeProject - The compose project name
 * @returns {string} Base app ID without instance suffix
 */
export function getBaseAppId(composeProject) {
  if (!composeProject) return composeProject;
  return composeProject.replace(/-\d+$/, '');
}
