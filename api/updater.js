import Docker from 'dockerode';

const socketPath = process.env.DOCKER_SOCKET || '/var/run/docker.sock';
const docker = new Docker({ socketPath });

setInterval(async () => {
  console.log('[UPDATER] Checking for updates...');
  try {
    await new Promise((resolve, reject) => {
      docker.pull('ghcr.io/besoeasy/yantra', (err, stream) => {
        if (err) return reject(err);
        docker.modem.followProgress(stream, (err) => {
          if (err) reject(err);
          else resolve();
        });
      });
    });
    console.log('[UPDATER] Yantra image updated. Will use on next restart.');
  } catch (err) {
    console.error('[UPDATER] Update failed:', err.message);
  }
}, 12 * 60 * 60 * 1000);

export default {};
