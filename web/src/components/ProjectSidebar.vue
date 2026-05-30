<template>
  <aside class="sidebar">
    <div class="sidebar-title">部署项目</div>
    <el-scrollbar>
      <div
        v-for="proj in projects"
        :key="proj.id"
        class="project-item"
        :class="{ active: proj.id === selectedId }"
        @click="$emit('select-project', proj)"
      >
        <div class="proj-name-row">
          <span class="proj-name">{{ proj.name }}</span>
          <el-badge :value="proj.environments?.length || 0" type="info" class="env-badge" />
        </div>
        <div class="proj-id">{{ proj.id }}</div>
      </div>
      <el-empty v-if="projects.length === 0" description="未加载到项目配置" :image-size="60" />
    </el-scrollbar>
  </aside>
</template>

<script setup lang="ts">
defineProps<{
  projects: any[]
  selectedId: string
}>()

defineEmits<{
  (e: 'select-project', project: any): void
}>()
</script>

<style scoped>
.sidebar {
  width: 260px;
  background-color: #151922;
  border-right: 1px solid rgba(255, 255, 255, 0.08);
  display: flex;
  flex-direction: column;
}

.sidebar-title {
  padding: 16px 20px;
  font-size: 13px;
  font-weight: 600;
  color: #8a99ad;
  text-transform: uppercase;
  letter-spacing: 1px;
}

.project-item {
  padding: 16px 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.03);
  cursor: pointer;
  transition: all 0.2s;
}

.project-item:hover {
  background-color: rgba(255, 255, 255, 0.03);
}

.project-item.active {
  background: linear-gradient(90deg, rgba(0, 180, 216, 0.1) 0%, rgba(0, 0, 0, 0) 100%);
  border-left: 3px solid #00b4d8;
}

.proj-name-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.proj-name {
  font-size: 15px;
  font-weight: 600;
  color: #ffffff;
}

.proj-id {
  font-size: 12px;
  color: #8a99ad;
  margin-top: 4px;
}

.env-badge {
  margin-left: auto;
}
</style>
