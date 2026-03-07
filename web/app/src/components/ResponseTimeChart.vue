<template>
  <div class="relative w-full" style="height: 300px;">
    <div v-if="loading" class="absolute inset-0 flex items-center justify-center bg-background/50">
      <Loading />
    </div>
    <div v-else-if="error" class="absolute inset-0 flex items-center justify-center text-muted-foreground">
      {{ error }}
    </div>
    <Line v-else :data="chartData" :options="chartOptions" />
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { Line } from 'vue-chartjs'
import { Chart as ChartJS, CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend, Filler, TimeScale } from 'chart.js'
import annotationPlugin from 'chartjs-plugin-annotation'
import 'chartjs-adapter-date-fns'
import { generatePrettyTimeDifference } from '@/utils/time'
import Loading from './Loading.vue'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend, Filler, TimeScale, annotationPlugin)

const props = defineProps({
  endpointKey: {
    type: String,
    required: true
  },
  duration: {
    type: String,
    required: true,
    validator: (value) => ['1h', '24h', '7d', '30d'].includes(value)
  },
  serverUrl: {
    type: String,
    default: '..'
  },
  events: {
    type: Array,
    default: () => []
  },
  metrics: {
    type: Array,
    default: () => []
  }
})

const loading = ref(true)
const error = ref(null)
const isDark = ref(document.documentElement.classList.contains('dark'))
const hoveredEventIndex = ref(null)

// Data model: supports both single response-time and multi-metric series
const dataType = ref('response-time') // 'metric' or 'response-time'
const series = ref([]) // [{name, unit, timestamps, values}]
// For response-time fallback
const rtTimestamps = ref([])
const rtValues = ref([])

// Fixed color pool to avoid color drift on refresh
const COLOR_POOL = [
  { border: 'rgb(59, 130, 246)',  bg: 'rgba(59, 130, 246, 0.1)',  borderDark: 'rgb(96, 165, 250)',  bgDark: 'rgba(96, 165, 250, 0.1)' },
  { border: 'rgb(16, 185, 129)',  bg: 'rgba(16, 185, 129, 0.1)',  borderDark: 'rgb(52, 211, 153)',  bgDark: 'rgba(52, 211, 153, 0.1)' },
  { border: 'rgb(245, 158, 11)',  bg: 'rgba(245, 158, 11, 0.1)',  borderDark: 'rgb(251, 191, 36)',  bgDark: 'rgba(251, 191, 36, 0.1)' },
  { border: 'rgb(239, 68, 68)',   bg: 'rgba(239, 68, 68, 0.1)',   borderDark: 'rgb(248, 113, 113)', bgDark: 'rgba(248, 113, 113, 0.1)' },
  { border: 'rgb(139, 92, 246)',  bg: 'rgba(139, 92, 246, 0.1)',  borderDark: 'rgb(167, 139, 250)', bgDark: 'rgba(167, 139, 250, 0.1)' },
  { border: 'rgb(236, 72, 153)',  bg: 'rgba(236, 72, 153, 0.1)',  borderDark: 'rgb(244, 114, 182)', bgDark: 'rgba(244, 114, 182, 0.1)' },
  { border: 'rgb(20, 184, 166)',  bg: 'rgba(20, 184, 166, 0.1)',  borderDark: 'rgb(45, 212, 191)',  bgDark: 'rgba(45, 212, 191, 0.1)' },
  { border: 'rgb(99, 102, 241)',  bg: 'rgba(99, 102, 241, 0.1)',  borderDark: 'rgb(129, 140, 248)', bgDark: 'rgba(129, 140, 248, 0.1)' },
]

// Helper function to get color for unhealthy events
const getEventColor = () => {
  return 'rgba(239, 68, 68, 0.8)'
}

// Filter events based on selected duration and calculate durations
const filteredEvents = computed(() => {
  if (!props.events || props.events.length === 0) {
    return []
  }

  const now = new Date()
  let fromTime
  switch (props.duration) {
    case '1h':
      fromTime = new Date(now.getTime() - 60 * 60 * 1000)
      break
    case '24h':
      fromTime = new Date(now.getTime() - 24 * 60 * 60 * 1000)
      break
    case '7d':
      fromTime = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000)
      break
    case '30d':
      fromTime = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000)
      break
    default:
      return []
  }

  const unhealthyEvents = []
  for (let i = 0; i < props.events.length; i++) {
    const event = props.events[i]
    if (event.type !== 'UNHEALTHY') continue

    const eventTime = new Date(event.timestamp)
    if (eventTime < fromTime || eventTime > now) continue

    let duration = null
    let isOngoing = false
    if (i + 1 < props.events.length) {
      const nextEvent = props.events[i + 1]
      duration = generatePrettyTimeDifference(nextEvent.timestamp, event.timestamp)
    } else {
      duration = generatePrettyTimeDifference(now, event.timestamp)
      isOngoing = true
    }

    unhealthyEvents.push({
      ...event,
      duration,
      isOngoing
    })
  }

  return unhealthyEvents
})

// Compute global max Y across all datasets for annotation positioning
const globalMaxY = computed(() => {
  if (dataType.value === 'metric') {
    let max = 0
    for (const s of series.value) {
      for (const v of s.values) {
        if (v > max) max = v
      }
    }
    return max
  }
  return rtValues.value.length > 0 ? Math.max(...rtValues.value) : 0
})

const chartData = computed(() => {
  if (dataType.value === 'metric') {
    // Multi-metric mode
    if (series.value.length === 0) {
      return { labels: [], datasets: [] }
    }
    // Use the longest timestamps array as labels (each series has its own timestamps)
    // For Chart.js, we use x/y data points per dataset for independent timestamps
    const datasets = series.value.map((s, idx) => {
      const color = COLOR_POOL[idx % COLOR_POOL.length]
      const label = `${s.name}${s.unit ? ` (${s.unit})` : ''}`
      const data = s.timestamps.map((ts, i) => ({
        x: new Date(ts),
        y: s.values[i]
      }))
      return {
        label,
        data,
        borderColor: isDark.value ? color.borderDark : color.border,
        backgroundColor: isDark.value ? color.bgDark : color.bg,
        borderWidth: 2,
        pointRadius: 2,
        pointHoverRadius: 4,
        tension: 0.1,
        fill: false,
        _unit: s.unit || ''
      }
    })
    return { datasets }
  }

  // Response time mode (single dataset)
  if (rtTimestamps.value.length === 0) {
    return { labels: [], datasets: [] }
  }
  const labels = rtTimestamps.value.map(ts => new Date(ts))
  return {
    labels,
    datasets: [{
      label: 'Response Time (ms)',
      data: rtValues.value,
      borderColor: isDark.value ? 'rgb(96, 165, 250)' : 'rgb(59, 130, 246)',
      backgroundColor: isDark.value ? 'rgba(96, 165, 250, 0.1)' : 'rgba(59, 130, 246, 0.1)',
      borderWidth: 2,
      pointRadius: 2,
      pointHoverRadius: 4,
      tension: 0.1,
      fill: true,
      _unit: 'ms'
    }]
  }
})

const chartOptions = computed(() => {
  // eslint-disable-next-line no-unused-vars
  const _ = hoveredEventIndex.value

  const maxY = globalMaxY.value
  const midY = maxY / 2
  const isMultiMetric = dataType.value === 'metric' && series.value.length > 0
  const showLegend = isMultiMetric && series.value.length > 1

  return {
    responsive: true,
    maintainAspectRatio: false,
    interaction: {
      mode: 'index',
      intersect: false
    },
    plugins: {
      title: {
        display: true,
        text: isMultiMetric
          ? 'Core Monitoring Indicators'
          : 'Response Time Trend',
        align: 'start',
        color: isDark.value ? '#f9fafb' : '#111827',
        font: {
          size: 16,
          weight: 'bold'
        },
        padding: {
          bottom: 20
        }
      },
      legend: {
        display: showLegend,
        position: 'top',
        labels: {
          color: isDark.value ? '#d1d5db' : '#374151',
          usePointStyle: true,
          pointStyle: 'circle',
          padding: 16
        }
      },
      tooltip: {
        backgroundColor: isDark.value ? 'rgba(31, 41, 55, 0.95)' : 'rgba(255, 255, 255, 0.95)',
        titleColor: isDark.value ? '#f9fafb' : '#111827',
        bodyColor: isDark.value ? '#d1d5db' : '#374151',
        borderColor: isDark.value ? '#4b5563' : '#e5e7eb',
        borderWidth: 1,
        padding: 12,
        displayColors: showLegend,
        callbacks: {
          title: (tooltipItems) => {
            if (tooltipItems.length > 0) {
              const date = new Date(tooltipItems[0].parsed.x)
              return date.toLocaleString()
            }
            return ''
          },
          label: (context) => {
            const value = context.parsed.y
            const unit = context.dataset._unit || ''
            const name = context.dataset.label || ''
            if (unit) {
              return `${name}: ${value}${unit}`
            }
            return `${name}: ${value}`
          }
        }
      },
      annotation: {
        annotations: filteredEvents.value.reduce((acc, event, index) => {
          const eventTimestamp = new Date(event.timestamp).getTime()
          let closestValue = 0

          // Find closest value across ALL datasets for annotation positioning
          if (dataType.value === 'metric') {
            for (const s of series.value) {
              if (s.timestamps.length > 0) {
                const closestIndex = s.timestamps.reduce((closest, ts, idx) => {
                  const currentDistance = Math.abs(ts - eventTimestamp)
                  const closestDistance = Math.abs(s.timestamps[closest] - eventTimestamp)
                  return currentDistance < closestDistance ? idx : closest
                }, 0)
                if (s.values[closestIndex] > closestValue) {
                  closestValue = s.values[closestIndex]
                }
              }
            }
          } else if (rtTimestamps.value.length > 0 && rtValues.value.length > 0) {
            const closestIndex = rtTimestamps.value.reduce((closest, ts, idx) => {
              const tsTime = new Date(ts).getTime()
              const currentDistance = Math.abs(tsTime - eventTimestamp)
              const closestDistance = Math.abs(new Date(rtTimestamps.value[closest]).getTime() - eventTimestamp)
              return currentDistance < closestDistance ? idx : closest
            }, 0)
            closestValue = rtValues.value[closestIndex]
          }

          const position = closestValue <= midY ? 'end' : 'start'

          acc[`event-${index}`] = {
            type: 'line',
            xMin: new Date(event.timestamp),
            xMax: new Date(event.timestamp),
            borderColor: getEventColor(),
            borderWidth: 1,
            borderDash: [5, 5],
            enter() {
              hoveredEventIndex.value = index
            },
            leave() {
              hoveredEventIndex.value = null
            },
            label: {
              display: () => hoveredEventIndex.value === index,
              content: [event.isOngoing ? `Status: ONGOING` : `Status: RESOLVED`, `Unhealthy for ${event.duration}`, `Started at ${new Date(event.timestamp).toLocaleString()}`],
              backgroundColor: getEventColor(),
              color: '#ffffff',
              font: {
                size: 11
              },
              padding: 6,
              position
            }
          }
          return acc
        }, {})
      }
    },
    scales: {
      x: {
        type: 'time',
        time: {
          unit: props.duration === '1h' ? 'minute' : props.duration === '24h' ? 'hour' : 'day',
          displayFormats: {
            minute: 'HH:mm',
            hour: 'MMM d, ha',
            day: 'MMM d'
          }
        },
        grid: {
          color: isDark.value ? 'rgba(75, 85, 99, 0.3)' : 'rgba(229, 231, 235, 0.8)',
          drawBorder: false
        },
        ticks: {
          color: isDark.value ? '#9ca3af' : '#6b7280',
          maxRotation: 0,
          autoSkipPadding: 20
        }
      },
      y: {
        beginAtZero: true,
        grid: {
          color: isDark.value ? 'rgba(75, 85, 99, 0.3)' : 'rgba(229, 231, 235, 0.8)',
          drawBorder: false
        },
        ticks: {
          color: isDark.value ? '#9ca3af' : '#6b7280'
        }
      }
    }
  }
})

const fetchData = async () => {
  loading.value = true
  error.value = null
  
  try {
    if (props.metrics && props.metrics.length > 0) {
      // Fetch multi-metric data from the series API
      dataType.value = 'metric'
      
      const metricResponse = await fetch(
        `${props.serverUrl}/api/v1/endpoints/${props.endpointKey}/metrics/${props.duration}`,
        { credentials: 'include' }
      )
      
      if (metricResponse.status === 200) {
        const metricData = await metricResponse.json()
        const rawSeries = metricData.series || []
        series.value = rawSeries.map(s => ({
          name: s.name,
          unit: s.unit || '',
          timestamps: s.timestamps || [],
          values: (s.values || []).map(v => {
            const num = parseFloat(v)
            return isNaN(num) ? 0 : num
          })
        }))
      } else {
        error.value = 'Failed to load metric data'
      }
    } else {
      // Fetch Response Time data
      dataType.value = 'response-time'
      series.value = []
      
      const responseTimeResponse = await fetch(
        `${props.serverUrl}/api/v1/endpoints/${props.endpointKey}/response-times/${props.duration}/history`,
        { credentials: 'include' }
      )
      
      if (responseTimeResponse.status === 200) {
        const data = await responseTimeResponse.json()
        rtTimestamps.value = data.timestamps || []
        rtValues.value = data.values || []
      } else {
        error.value = 'Failed to load chart data'
      }
    }
  } catch (err) {
    error.value = 'Failed to load chart data'
    console.error('[ResponseTimeChart] Error:', err)
  } finally {
    loading.value = false
  }
}

watch(() => props.duration, () => {
  fetchData()
})

onMounted(() => {
  fetchData()
  const observer = new MutationObserver(() => {
    isDark.value = document.documentElement.classList.contains('dark')
  })
  observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
  onUnmounted(() => observer.disconnect())
})
</script>
