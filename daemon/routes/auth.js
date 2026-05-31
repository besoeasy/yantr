import { extractBearerToken, loadAuthConfig, saveAuthConfig, verifyYantrAuthToken } from "../auth.js";
import { nonEmptyStringSchema, publicKeySchema, sendError } from "../api.js";

const setupAdminSchema = {
  body: {
    type: "object",
    required: ["username", "publicKey"],
    additionalProperties: false,
    properties: {
      username: nonEmptyStringSchema,
      publicKey: publicKeySchema,
    },
  },
};

export default async function authRoutes(fastify) {
  fastify.get("/api/setup/status", async (_request, reply) => {
    const config = await loadAuthConfig();
    return reply.send({ success: true, configured: !!config });
  });

  fastify.post("/api/setup/admin", { schema: setupAdminSchema }, async (request, reply) => {
    const existing = await loadAuthConfig();
    if (existing) {
      return sendError(reply, 409, { code: "SETUP_ALREADY_CONFIGURED", message: "Yantr is already configured" });
    }

    const { username, publicKey } = request.body || {};
    try {
      const config = await saveAuthConfig({ username, publicKey });
      return reply.code(201).send({ success: true, configured: true, username: config.username });
    } catch (error) {
      return sendError(reply, 400, { code: "INVALID_SETUP_ADMIN_REQUEST", message: error.message });
    }
  });

  fastify.post("/api/auth/login", async (request, reply) => {
    const result = await verifyYantrAuthToken(extractBearerToken(request));

    if (!result.config) {
      return sendError(reply, 409, { code: "SETUP_REQUIRED", message: "Setup required" });
    }

    if (!result.ok) {
      return sendError(reply, 401, { code: String(result.reason || "UNAUTHORIZED").toUpperCase(), message: "Unauthorized" });
    }

    return reply.send({
      success: true,
      authenticated: true,
      user: {
        username: result.config.username,
        publicKey: result.publicKey,
      },
    });
  });
}