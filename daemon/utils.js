import { spawn } from "child_process";
import { sendError } from "./api.js";

/**
 * Helper function to spawn a process and capture output
 * @param {string} command - The command to execute
 * @param {string[]} args - Array of arguments to pass to the command
 * @param {Object} options - Options to pass to spawn (cwd, env, etc.)
 * @returns {Promise<{stdout: string, stderr: string, exitCode: number}>}
 */
export function spawnProcess(command, args, options = {}) {
  const { timeout, ...spawnOptions } = options;
  return new Promise((resolve, reject) => {
    const proc = spawn(command, args, {
      ...spawnOptions,
      env: spawnOptions.env || process.env,
    });

    let stdout = '';
    let stderr = '';
    let timedOut = false;

    const timer = timeout
      ? setTimeout(() => {
          timedOut = true;
          proc.kill('SIGKILL');
        }, timeout)
      : null;

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
      if (timer) clearTimeout(timer);
      if (timedOut) {
        reject(new Error(`Process timed out after ${timeout}ms`));
      } else {
        resolve({ stdout, stderr, exitCode: code });
      }
    });

    proc.on('error', (err) => {
      if (timer) clearTimeout(timer);
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
  constructor(message, statusCode = 500, code = "INTERNAL_SERVER_ERROR", details = null) {
    super(message);
    this.name = this.constructor.name;
    this.statusCode = statusCode;
    this.code = code;
    this.details = details;
    this.isOperational = true; // Distinguish operational errors from programming errors
    Error.captureStackTrace(this, this.constructor);
  }
}

/**
 * Bad request error (400)
 */
export class BadRequestError extends AppError {
  constructor(message, details = null, code = "BAD_REQUEST") {
    super(message, 400, code, details);
  }
}

/**
 * Not found error (404)
 */
export class NotFoundError extends AppError {
  constructor(message, details = null, code = "NOT_FOUND") {
    super(message, 404, code, details);
  }
}

/**
 * Conflict error (409)
 */
export class ConflictError extends AppError {
  constructor(message, details = null, code = "CONFLICT") {
    super(message, 409, code, details);
  }
}

/**
 * Docker-specific error
 */
export class DockerError extends AppError {
  constructor(message, details = null, code = "DOCKER_ERROR") {
    super(message, 500, code, details);
  }
}

/**
 * Fastify error handler — register with fastify.setErrorHandler(errorHandler)
 * @param {Error} err
 * @param {import('fastify').FastifyRequest} request
 * @param {import('fastify').FastifyReply} reply
 */
export function errorHandler(err, request, reply) {
  const timestamp = new Date().toISOString();
  console.error(`[${timestamp}] ❌ [ERROR] ${request.method} ${request.url}`);
  console.error(`[${timestamp}] ❌ [ERROR] ${err.stack || err.message}`);

  if (err.validation) {
    return sendError(reply, 400, {
      code: "VALIDATION_ERROR",
      message: "Request validation failed",
      details: err.validation.map((issue) => ({
        instancePath: issue.instancePath || "",
        schemaPath: issue.schemaPath || "",
        keyword: issue.keyword || "validation",
        message: issue.message || "Invalid value",
        params: issue.params || {},
      })),
    });
  }

  // Operational errors (AppError subclasses)
  if (err.isOperational) {
    return sendError(reply, err.statusCode, {
      code: err.code || "OPERATIONAL_ERROR",
      message: err.message,
      details: err.details,
    });
  }

  // Docker API errors
  if (err.statusCode && err.reason) {
    return sendError(reply, err.statusCode >= 400 && err.statusCode < 600 ? err.statusCode : 500, {
      code: "DOCKER_API_ERROR",
      message: err.reason || err.message,
      details: err.json || null,
    });
  }

  const isDevelopment = process.env.NODE_ENV === 'development';
  return sendError(reply, 500, {
    code: "INTERNAL_SERVER_ERROR",
    message: "Internal server error",
    details: isDevelopment ? { message: err.message, stack: err.stack } : null,
  });
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
