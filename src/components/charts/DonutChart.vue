<script setup>
import { computed } from 'vue'
import VueApexCharts from 'vue3-apexcharts'

const props = defineProps({
  series: {
    type: Array,
    default: () => []
  },
  labels: {
    type: Array,
    default: () => []
  },
  height: {
    type: [Number, String],
    default: 190
  },
  colors: {
    type: Array,
    default: () => undefined
  },
  valueFormatter: {
    type: Function,
    default: null
  },
  totalFormatter: {
    type: Function,
    default: null
  },
  donutLabel: {
    type: String,
    default: ''
  }
})

const chartOptions = computed(() => {
  const formatValue = (val) => {
    if (typeof props.valueFormatter === 'function') return props.valueFormatter(val)
    return `${Math.round(val)}`
  }

  return {
    chart: {
      type: 'donut',
      sparkline: { enabled: true }
    },
    labels: props.labels,
    colors: props.colors,
    dataLabels: { enabled: false },
    legend: { show: false },
    stroke: { width: 0 },
    tooltip: {
      y: {
        formatter: (val) => formatValue(val)
      }
    },
    plotOptions: {
      pie: {
        donut: {
          size: '72%',
          labels: {
            show: true,
            name: {
              show: true,
              fontSize: '12px',
              fontWeight: 700,
              offsetY: 18
            },
            value: {
              show: true,
              fontSize: '14px',
              fontWeight: 800,
              offsetY: -2,
              formatter: (val) => formatValue(val)
            },
            total: {
              show: true,
              label: props.donutLabel || 'Total',
              fontSize: '12px',
              fontWeight: 700,
              formatter: () => {
                if (typeof props.totalFormatter === 'function') return props.totalFormatter()
                const sum = (props.series || []).reduce((s, n) => s + (Number(n) || 0), 0)
                return formatValue(sum)
              }
            }
          }
        }
      }
    }
  }
})
</script>

<template>
  <VueApexCharts type="donut" :height="height" :options="chartOptions" :series="series" />
</template>
