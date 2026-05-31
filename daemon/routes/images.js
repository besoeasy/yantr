import path from "path";
import { readFile } from "fs/promises";
import { docker, log, appsDir } from "../shared.js";
import { idParamsSchema, sendError } from "../api.js";

export default async function imagesRoutes(fastify) {
  // GET /api/images
  fastify.get("/api/images", async (request, reply) => {
    try {
      const images = await docker.listImages();
      const containers = await docker.listContainers({ all: true });
      const usedImageIds = new Set(containers.map(c => c.ImageID));

      const formattedImages = images.map(image => {
        const isUsed = usedImageIds.has(image.Id);
        return {
          id: image.Id,
          shortId: image.Id.substring(7, 19),
          tags: image.RepoTags || ["<none>:<none>"],
          created: image.Created,
          size: (image.Size / (1024 * 1024)).toFixed(2),
          sizeBytes: image.Size,
          isUsed,
          containers: image.Containers || 0,
        };
      });

      const usedImages = formattedImages.filter(i => i.isUsed).sort((a, b) => b.sizeBytes - a.sizeBytes);
      const unusedImages = formattedImages.filter(i => !i.isUsed).sort((a, b) => b.sizeBytes - a.sizeBytes);
      const totalSize = formattedImages.reduce((s, i) => s + i.sizeBytes, 0);
      const unusedSize = unusedImages.reduce((s, i) => s + i.sizeBytes, 0);

      return reply.send({
        success: true,
        total: formattedImages.length,
        used: usedImages.length,
        unused: unusedImages.length,
        totalSize: (totalSize / (1024 * 1024)).toFixed(2),
        unusedSize: (unusedSize / (1024 * 1024)).toFixed(2),
        images: formattedImages,
        usedImages,
        unusedImages,
      });
    } catch (error) {
      log("error", "❌ [GET /api/images] Error:", error.message);
      return reply.code(500).send({ success: false, error: error.message });
    }
  });

  // GET /api/image-details/:appId
  fastify.get("/api/image-details/:appId", async (request, reply) => {
    try {
      const appId = request.params.appId;
      const composePath = path.join(appsDir, appId, "compose.yml");

      let composeContent;
      try { composeContent = await readFile(composePath, "utf-8"); }
      catch { return reply.code(404).send({ success: false, error: "App compose file not found" }); }

      const imageNames = [...new Set([...composeContent.matchAll(/image:\s*([^\s\n]+)/g)].map(m => m[1]))];
      if (imageNames.length === 0) return reply.send({ success: true, images: [] });

      const localImages = await docker.listImages();

      const imageDetails = await Promise.all(imageNames.map(async imageName => {
        const localImage = localImages.find(img => img.RepoTags?.some(tag => tag === imageName || tag.includes(imageName.split(":")[0])));

        if (localImage) {
          try {
            const info = await docker.getImage(localImage.Id).inspect();
            const createdDate = new Date(info.Created);
            const diffMs = Date.now() - createdDate;
            const diffDays = Math.floor(diffMs / 86400000);
            let relativeTime;
            if (diffDays === 0) { const h = Math.floor(diffMs / 3600000); relativeTime = h === 0 ? "just now" : `${h} hour${h !== 1 ? "s" : ""} ago`; }
            else if (diffDays === 1) relativeTime = "yesterday";
            else if (diffDays < 30) relativeTime = `${diffDays} day${diffDays !== 1 ? "s" : ""} ago`;
            else if (diffDays < 365) { const m = Math.floor(diffDays / 30); relativeTime = `${m} month${m !== 1 ? "s" : ""} ago`; }
            else { const y = Math.floor(diffDays / 365); relativeTime = `${y} year${y !== 1 ? "s" : ""} ago`; }

            return { name: imageName, id: localImage.Id, shortId: localImage.Id.substring(7, 19), tags: localImage.RepoTags || [imageName], digest: info.RepoDigests?.[0] || "N/A", created: info.Created, createdDate: createdDate.toLocaleDateString("en-US", { year: "numeric", month: "short", day: "numeric" }), relativeTime, architecture: info.Architecture || "unknown", os: info.Os || "unknown", size: (localImage.Size / (1024 * 1024)).toFixed(2), sizeBytes: localImage.Size, isLocal: true };
          } catch (err) {
            log("error", `❌ Error inspecting image ${imageName}:`, err.message);
          }
        }

        return { name: imageName, id: null, shortId: "Not pulled", tags: [imageName], digest: "N/A", created: null, createdDate: "Not available", relativeTime: "Image not pulled yet", architecture: "unknown", os: "unknown", size: "N/A", sizeBytes: 0, isLocal: false };
      }));

      return reply.send({ success: true, images: imageDetails.filter(Boolean) });
    } catch (error) {
      log("error", `❌ [GET /api/image-details] Error:`, error.message);
      return reply.code(500).send({ success: false, error: error.message });
    }
  });

  // DELETE /api/images/:id
  fastify.delete("/api/images/:id", { schema: { params: idParamsSchema } }, async (request, reply) => {
    log("info", `🗑️  [DELETE /api/images/:id] Remove request for image: ${request.params.id}`);
    try {
      const image = docker.getImage(request.params.id);
      const info = await image.inspect();
      await image.remove({ force: false });
      return reply.send({ success: true, message: "Image removed successfully", imageId: request.params.id, tags: info.RepoTags });
    } catch (error) {
      log("error", `❌ [DELETE /api/images/:id] Error:`, error.message);
      return sendError(reply, 500, { code: "IMAGE_REMOVE_FAILED", message: "Failed to remove image", details: { cause: error.message } });
    }
  });
}
