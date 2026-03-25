<template>
  <div class="ip-view">
    <ProTable
      ref="proTableRef"
      api="/asset/ip/list"
      statApi="/asset/ip/stat"
      batchDeleteApi="/asset/ip/batchDelete"
      exportApi="/asset/ip/list"
      rowKey="ip"
      :columns="ipColumns"
      :searchItems="ipSearchItems"
      :statLabels="statLabels"
      selection
      @data-changed="$emit('data-changed')"
    >
      <template #toolbar-right>
        <el-button type="danger" plain @click="handleClear">{{ $t('asset.clearData') }}</el-button>
      </template>

      <template #ip="{ row }">
        <div class="ip-cell">
          <span class="ip-text">{{ row.ip }}</span>
          <el-tag v-if="row.isNew" type="success" size="small" effect="dark" class="new-tag">{{ $t('common.new') }}</el-tag>
        </div>
      </template>

      <template #location="{ row }">
        {{ row.location || '-' }}
      </template>

      <template #ports="{ row }">
        <div class="port-list">
          <el-tag v-for="port in (row.ports || []).slice(0, 10)" :key="port.port" size="small" :type="getPortType(port.service)" class="port-tag">
            {{ port.port }}<span v-if="port.service">/{{ port.service }}</span>
          </el-tag>
          <span v-if="(row.ports || []).length > 10" class="more-ports">+{{ (row.ports || []).length - 10 }}</span>
        </div>
      </template>

      <template #domains="{ row }">
        <div v-if="row.domains && row.domains.length > 0" class="domain-list">
          <div v-for="domain in row.domains.slice(0, 5)" :key="domain" class="domain-item">{{ domain }}</div>
          <div v-if="row.domains.length > 5" class="more-domains">+{{ row.domains.length - 5 }} {{ $t('common.more') }}...</div>
        </div>
        <span v-else>-</span>
      </template>

      <template #org="{ row }">
        {{ row.orgName || $t('common.defaultOrganization') }}
      </template>

      <template #operation="{ row }">
        <el-button type="primary" link size="small" @click="showDetail(row)">{{ $t('common.detail') }}</el-button>
        <el-button type="danger" link size="small" @click="handleDelete(row)">{{ $t('common.delete') }}</el-button>
      </template>
    </ProTable>

    <!-- 详情侧边栏 -->
    <el-drawer v-model="detailVisible" :title="$t('ip.ipDetail')" size="50%" direction="rtl">
      <el-descriptions v-if="currentIP" :column="2" border>
        <el-descriptions-item :label="$t('ip.ipAddress')">{{ currentIP.ip }}</el-descriptions-item>
        <el-descriptions-item :label="$t('ip.location')">{{ currentIP.location || '-' }}</el-descriptions-item>
        <el-descriptions-item :label="$t('ip.openPorts')" :span="2">
          <div class="detail-ports">
            <el-tag v-for="port in (currentIP.ports || [])" :key="port.port" size="small" style="margin: 2px">
              {{ port.port }}<span v-if="port.service">/{{ port.service }}</span>
            </el-tag>
          </div>
        </el-descriptions-item>
        <el-descriptions-item :label="$t('ip.relatedDomains')" :span="2">
          <div v-if="currentIP.domains && currentIP.domains.length > 0">
            <el-tag v-for="domain in currentIP.domains" :key="domain" type="info" size="small" style="margin: 2px">{{ domain }}</el-tag>
          </div>
          <span v-else>-</span>
        </el-descriptions-item>
      </el-descriptions>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/api/request'
import { clearAssets } from '@/api/asset'
import ProTable from '@/components/common/ProTable.vue'

const { t } = useI18n()
const emit = defineEmits(['data-changed'])

const proTableRef = ref(null)
const organizations = ref([])
const detailVisible = ref(false)
const currentIP = ref(null)

const statLabels = computed(() => ({
  total: t('ip.totalIPs'),
  portCount: t('ip.openPorts'),
  serviceCount: t('ip.serviceTypes'),
  newCount: t('ip.newIPs')
}))

const ipColumns = computed(() => [
  { label: t('ip.ipAddress'), prop: 'ip', slot: 'ip', width: 180 },
  { label: t('ip.location'), prop: 'location', slot: 'location', width: 200 },
  { label: t('ip.openPorts'), prop: 'ports', slot: 'ports', minWidth: 300 },
  { label: t('ip.relatedDomains'), prop: 'domains', slot: 'domains', minWidth: 200 },
  { label: t('ip.organization'), prop: 'orgName', slot: 'org', width: 120 },
  { label: t('common.updateTime'), prop: 'updateTime', width: 160 },
  { label: t('common.operation'), slot: 'operation', width: 100, fixed: 'right' }
])

const ipSearchItems = computed(() => [
  { label: t('ip.ipAddress'), prop: 'ip', type: 'input' },
  { label: t('ip.port'), prop: 'port', type: 'input' },
  { label: t('ip.service'), prop: 'service', type: 'input' },
  { label: t('ip.location'), prop: 'location', type: 'input' },
  {
    label: t('ip.organization'),
    prop: 'orgId',
    type: 'select',
    options: [
      { label: t('common.allOrganizations'), value: '' },
      ...organizations.value.map(org => ({ label: org.name, value: org.id }))
    ]
  }
])

async function loadOrganizations() {
  try {
    const res = await request.post('/organization/list', { page: 1, pageSize: 100 })
    if (res.code === 0) organizations.value = res.list || []
  } catch (e) {
    console.error(e)
  }
}

function showDetail(row) {
  currentIP.value = row
  detailVisible.value = true
}

async function handleDelete(row) {
  try {
    await ElMessageBox.confirm(t('ip.confirmDeleteIP'), t('common.tip'), { type: 'warning' })
    const res = await request.post('/asset/ip/delete', { ip: row.ip })
    if (res.code === 0) {
      ElMessage.success(t('common.deleteSuccess'))
      proTableRef.value?.loadData()
      emit('data-changed')
    }
  } catch (e) {
    // cancelled
  }
}

async function handleClear() {
  try {
    await ElMessageBox.confirm(t('asset.confirmClearAll'), t('common.warning'), {
      type: 'error',
      confirmButtonText: t('asset.confirmClearBtn'),
      cancelButtonText: t('common.cancel')
    })
    const res = await clearAssets()
    if (res.code === 0) {
      ElMessage.success(res.msg || t('asset.clearSuccess'))
      proTableRef.value?.loadData()
      emit('data-changed')
    } else {
      ElMessage.error(res.msg || t('asset.clearFailed'))
    }
  } catch (e) {
    if (e !== 'cancel') {
      console.error('清空资产失败:', e)
      ElMessage.error(t('asset.clearFailed'))
    }
  }
}

function getPortType(service) {
  if (!service) return 'info'
  if (['http', 'https'].includes(service)) return 'success'
  if (['ssh', 'ftp', 'telnet'].includes(service)) return 'warning'
  if (['mysql', 'redis', 'mongodb'].includes(service)) return 'danger'
  return 'info'
}

function refresh() {
  proTableRef.value?.loadData()
}

function handleWorkspaceChanged() {
  loadOrganizations()
}

onMounted(() => {
  loadOrganizations()
  window.addEventListener('workspace-changed', handleWorkspaceChanged)
})

onUnmounted(() => {
  window.removeEventListener('workspace-changed', handleWorkspaceChanged)
})

defineExpose({ refresh })
</script>

<style scoped lang="scss">
.ip-view {
  height: 100%;
  .ip-cell { display: flex; align-items: center; gap: 8px;
    .ip-text { font-family: monospace; font-weight: 500; }
  }
  .port-list { display: flex; flex-wrap: wrap; gap: 4px;
    .port-tag { font-family: monospace; }
    .more-ports { color: var(--el-text-color-secondary); font-size: 12px; line-height: 22px; }
  }
  .domain-list {
    .domain-item { font-family: monospace; font-size: 12px; line-height: 1.6; color: var(--el-text-color-regular);
      &:hover { color: var(--el-color-primary); }
    }
    .more-domains { font-size: 12px; color: var(--el-text-color-secondary); cursor: pointer;
      &:hover { color: var(--el-color-primary); }
    }
  }
  .detail-ports { max-height: 150px; overflow-y: auto; }
}
</style>
