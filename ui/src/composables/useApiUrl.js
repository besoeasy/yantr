import { ref } from 'vue'

export function useApiUrl() {
  const apiUrl = ref(window.VITE_API_URL || '')

  return { apiUrl }
}
