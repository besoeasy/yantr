export function createErrorResponse({ code, message, details = null }) {
  const payload = {
    success: false,
    code,
    message,
    error: message,
  };

  if (details !== null && details !== undefined) {
    payload.details = details;
  }

  return payload;
}

export function sendError(reply, statusCode, { code, message, details = null }) {
  return reply.code(statusCode).send(createErrorResponse({ code, message, details }));
}

export const nonEmptyStringSchema = {
  type: "string",
  minLength: 1,
};

export const optionalStringSchema = {
  type: "string",
};

export const positiveIntegerSchema = {
  type: "integer",
  minimum: 1,
};

export const nonNegativeIntegerSchema = {
  type: "integer",
  minimum: 0,
};

export const booleanSchema = {
  type: "boolean",
};

export const publicKeySchema = {
  type: "string",
  pattern: "^[0-9a-fA-F]{66}$",
};

export const scalarValueSchema = {
  anyOf: [
    { type: "string" },
    { type: "number" },
    { type: "boolean" },
    { type: "null" },
  ],
};

export const stringOrIntegerSchema = {
  anyOf: [
    { type: "string", minLength: 1 },
    { type: "integer", minimum: 0 },
  ],
};

export const scalarMapSchema = {
  type: "object",
  additionalProperties: scalarValueSchema,
};

export const idParamsSchema = {
  type: "object",
  required: ["id"],
  additionalProperties: false,
  properties: {
    id: nonEmptyStringSchema,
  },
};

export const nameParamsSchema = {
  type: "object",
  required: ["name"],
  additionalProperties: false,
  properties: {
    name: nonEmptyStringSchema,
  },
};

export const projectIdParamsSchema = {
  type: "object",
  required: ["projectId"],
  additionalProperties: false,
  properties: {
    projectId: nonEmptyStringSchema,
  },
};