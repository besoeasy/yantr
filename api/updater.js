import Docker from "dockerode";

const socketPath = process.env.DOCKER_SOCKET || "/var/run/docker.sock";
const docker = new Docker({ socketPath });

const IMAGE_NAME = "ghcr.io/besoeasy/yantra";

// Cleanup old Yantra images, keeping only the latest
export async function cleanupOldYantraImages() {
  try {
    console.log("[UPDATER] 🧹 Checking for old Yantra images to cleanup...");

    const images = await docker.listImages({
      filters: { reference: [IMAGE_NAME] },
    });

    if (images.length <= 1) {
      console.log("[UPDATER] ✅ No old images to cleanup");
      return;
    }

    // Sort images by creation date (newest first)
    images.sort((a, b) => b.Created - a.Created);

    // Keep the newest image, remove the rest
    const imagesToRemove = images.slice(1);

    for (const image of imagesToRemove) {
      try {
        const imageId = image.Id;
        console.log(`[UPDATER] 🗑️  Removing old image: ${imageId.substring(0, 12)}`);
        await docker.getImage(imageId).remove({ force: false });
        console.log(`[UPDATER] ✅ Removed old image: ${imageId.substring(0, 12)}`);
      } catch (err) {
        // Image might be in use or already removed, that's okay
        console.log(`[UPDATER] ⚠️  Could not remove image: ${err.message}`);
      }
    }

    console.log(`[UPDATER] 🎉 Cleanup complete. Kept 1 image, removed ${imagesToRemove.length}`);
  } catch (err) {
    console.error("[UPDATER] ❌ Failed to cleanup old images:", err.message);
  }
}

async function checkForUpdates() {
  console.log("[UPDATER] Checking for updates...");
  try {
    // Pull the latest image
    await new Promise((resolve, reject) => {
      docker.pull(IMAGE_NAME, (err, stream) => {
        if (err) return reject(err);
        docker.modem.followProgress(stream, (err) => {
          if (err) reject(err);
          else resolve();
        });
      });
    });

    // Check how many yantra images we have
    const images = await docker.listImages({
      filters: { reference: [IMAGE_NAME] },
    });

    console.log(`[UPDATER] Found ${images.length} Yantra image(s)`);

    // If we have 2+ images (old + new), a new version was pulled
    if (images.length > 1) {
      console.log("[UPDATER] 🎉 New version detected! Restarting to apply update...");
      // Exit the process - Docker will restart with the new image
      // and the startup cleanup will remove old images
      process.exit(0);
    } else {
      console.log("[UPDATER] Already on latest version");
    }
  } catch (err) {
    console.error("[UPDATER] Update check failed:", err.message);
  }
}

// Check for updates every hour
setInterval(checkForUpdates, 1 * 60 * 60 * 1000);

// Run cleanup on startup to remove any old images left from previous updates
cleanupOldYantraImages();
