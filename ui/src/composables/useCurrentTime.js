import { ref, onMounted, onUnmounted } from 'vue'

export function useCurrentTime(intervalMs = 1000) {
  const currentTime = ref(Date.now())
  let timeInterval = null

  onMounted(() => {
    timeInterval = setInterval(() => {
      currentTime.value = Date.now()
    }, intervalMs)
  })

  onUnmounted(() => {
    if (timeInterval) {
      clearInterval(timeInterval)
    }
  })

  return { currentTime }
}
