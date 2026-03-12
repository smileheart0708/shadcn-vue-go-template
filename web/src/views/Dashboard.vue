<script setup lang="ts">
import SectionCards from '@/components/SectionCards.vue'
import DataTable from '@/components/DataTable.vue'

const data = [
  {
    id: 1,
    header: "Introduction",
    type: "Overview",
    status: "Done",
    target: "1000",
    limit: "500",
    reviewer: "Eddie Lake",
  },
  {
    id: 2,
    header: "Background",
    type: "Context",
    status: "In Progress",
    target: "2500",
    limit: "1000",
    reviewer: "Jamik Tashpulatov",
  },
  {
    id: 3,
    header: "Methodology",
    type: "Approach",
    status: "In Progress",
    target: "3000",
    limit: "1500",
    reviewer: "Assign reviewer",
  },
  {
    id: 4,
    header: "Results",
    type: "Findings",
    status: "In Progress",
    target: "5000",
    limit: "2500",
    reviewer: "Assign reviewer",
  },
  {
    id: 5,
    header: "Discussion",
    type: "Analysis",
    status: "In Progress",
    target: "4000",
    limit: "2000",
    reviewer: "Assign reviewer",
  },
  {
    id: 6,
    header: "Conclusion",
    type: "Summary",
    status: "In Progress",
    target: "1500",
    limit: "750",
    reviewer: "Assign reviewer",
  },
  {
    id: 7,
    header: "References",
    type: "Bibliography",
    status: "In Progress",
    target: "2000",
    limit: "1000",
    reviewer: "Assign reviewer",
  },
  {
    id: 8,
    header: "Appendix",
    type: " supplementary",
    status: "In Progress",
    target: "3000",
    limit: "1500",
    reviewer: "Assign reviewer",
  },
]
</script>

<template>
  <div class="flex flex-1 flex-col gap-4 p-4 lg:gap-6 lg:p-6">
    <SectionCards />
    <DataTable :data="data" />
  </div>
</template>
