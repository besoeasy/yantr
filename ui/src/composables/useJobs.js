import { ref } from 'vue';
import { useApiUrl } from './useApiUrl';

export function useJobs() {
  const { apiUrl } = useApiUrl();
  const loading = ref(false);
  const error = ref(null);

  async function fetchJob(jobId) {
    if (!jobId) return null;
    try {
      const res = await fetch(`${apiUrl.value}/api/jobs/${jobId}`);
      if (!res.ok) return null;
      const data = await res.json();
      return data.success ? data.job : null;
    } catch (err) {
      console.warn('[jobs] failed to fetch job:', err);
      return null;
    }
  }

  async function fetchActiveJob(target) {
    if (!target) return null;
    try {
      const res = await fetch(`${apiUrl.value}/api/jobs/active?target=${encodeURIComponent(target)}`);
      if (!res.ok) return null;
      const data = await res.json();
      return data.active && data.job ? data.job : null;
    } catch (err) {
      console.warn('[jobs] failed to check active job:', err);
      return null;
    }
  }

  async function pollJobUntilDone(jobId, onProgress = null, maxAttempts = 900) {
    let attempts = 0;
    while (attempts < maxAttempts) {
      attempts++;
      const job = await fetchJob(jobId);
      if (!job) {
        await new Promise((resolve) => setTimeout(resolve, 2000));
        continue;
      }

      if (onProgress && typeof onProgress === 'function') {
        onProgress(job);
      }

      if (job.status === 'completed') {
        return { success: true, job };
      }
      if (job.status === 'failed') {
        return { success: false, job, error: job.error || 'Operation failed' };
      }

      // Wait 2s before next poll
      await new Promise((resolve) => setTimeout(resolve, 2000));
    }

    return { success: false, error: 'Job polling timed out' };
  }

  return {
    fetchJob,
    fetchActiveJob,
    pollJobUntilDone,
    loading,
    error,
  };
}
