<template>
  <div class="cron-task-page">
    <!-- 操作栏 -->
    <el-card class="action-card">
      <el-button type="primary" @click="showCreateDialog">
        <el-icon><Plus /></el-icon>{{ $t('cronTask.newCronTask') }}
      </el-button>
      <el-button @click="loadData">
        <el-icon><Refresh /></el-icon>{{ $t('common.refresh') }}
      </el-button>
      <el-button 
        type="danger" 
        :disabled="selectedRows.length === 0"
        @click="handleBatchDelete"
      >
        <el-icon><Delete /></el-icon>{{ $t('common.batchDelete') }} {{ selectedRows.length > 0 ? `(${selectedRows.length})` : '' }}
      </el-button>
    </el-card>

    <!-- 数据表格 -->
    <el-card class="table-card">
      <el-table 
        :data="tableData" 
        v-loading="loading" 
        stripe
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="50" />
        <el-table-column prop="name" :label="$t('cronTask.cronTaskName')" min-width="140" />
        <el-table-column prop="taskName" :label="$t('cronTask.relatedTask')" min-width="140">
          <template #default="{ row }">
            <span class="task-link" @click="goToTask(row)">{{ row.taskName }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="targetShort" :label="$t('cronTask.scanTarget')" min-width="180" show-overflow-tooltip />
        <el-table-column :label="$t('cronTask.scheduleType')" width="180">
          <template #default="{ row }">
            <div v-if="row.scheduleType === 'cron'">
              <el-tag type="primary" size="small">{{ $t('cronTask.cronExec').split(' ')[0] }}</el-tag>
              <el-tooltip :content="getCronDescription(row.cronSpec)" placement="top">
                <code class="cron-code">{{ row.cronSpec }}</code>
              </el-tooltip>
            </div>
            <div v-else>
              <el-tag type="warning" size="small">{{ $t('cronTask.onceExec').split(' ')[0] }}</el-tag>
              <span class="schedule-time">{{ row.scheduleTime }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column :label="$t('cronTask.status')" width="80">
          <template #default="{ row }">
            <el-switch
              v-model="row.status"
              active-value="enable"
              inactive-value="disable"
              @change="handleToggle(row)"
            />
          </template>
        </el-table-column>
        <el-table-column prop="nextRunTime" :label="$t('cronTask.nextRunTime')" width="160">
          <template #default="{ row }">
            <span v-if="row.status === 'enable' && row.nextRunTime">{{ row.nextRunTime }}</span>
            <span v-else class="text-muted">{{ row.status === 'disable' ? $t('cronTask.disabled') : '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="lastRunTime" :label="$t('cronTask.lastRunTime')" width="160">
          <template #default="{ row }">
            {{ row.lastRunTime || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="runCount" :label="$t('cronTask.runCount')" width="90">
          <template #default="{ row }">
            <el-tag type="info" size="small">{{ row.runCount || 0 }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="$t('common.operation')" width="200" fixed="right">
          <template #default="{ row }">
            <el-button type="success" link size="small" @click="handleRunNow(row)">
              <el-icon><VideoPlay /></el-icon>{{ $t('cronTask.runNow') }}
            </el-button>
            <el-button type="primary" link size="small" @click="handleEdit(row)">
              <el-icon><Edit /></el-icon>{{ $t('common.edit') }}
            </el-button>
            <el-button type="danger" link size="small" @click="handleDelete(row)">
              <el-icon><Delete /></el-icon>{{ $t('common.delete') }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        :page-sizes="[20, 50, 100]"
        layout="total, sizes, prev, pager, next"
        class="pagination"
        @size-change="loadData"
        @current-change="loadData"
      />
    </el-card>

    <!-- 新建/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? $t('cronTask.editCronTask') : $t('cronTask.newCronTask')" width="1000px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="110px">
        <el-form-item :label="$t('cronTask.cronTaskName')" prop="name">
          <el-input v-model="form.name" :placeholder="$t('cronTask.pleaseEnterName')" />
        </el-form-item>
        
        <el-form-item :label="$t('cronTask.relatedTask')" prop="mainTaskId">
          <el-select 
            v-model="form.mainTaskId" 
            :placeholder="$t('cronTask.pleaseSelectTask')" 
            style="width: 100%" 
            filterable
            @change="onTaskSelect"
          >
            <el-option 
              v-for="task in taskList" 
              :key="task.taskId" 
              :label="task.name" 
              :value="task.taskId"
            >
              <div class="task-option">
                <span class="task-name">{{ task.name }}</span>
                <span class="task-target">{{ truncateTarget(task.target) }}</span>
              </div>
            </el-option>
          </el-select>
        </el-form-item>

        <el-form-item :label="$t('cronTask.scanTarget')" prop="target">
          <el-input 
            v-model="form.target" 
            type="textarea" 
            :rows="4" 
            :placeholder="$t('cronTask.targetPlaceholder')"
          />
          <div class="form-hint">{{ $t('cronTask.targetHint') }}</div>
        </el-form-item>

        <el-form-item :label="$t('cronTask.scheduleType')" prop="scheduleType">
          <el-radio-group v-model="form.scheduleType">
            <el-radio label="cron">{{ $t('cronTask.cronExec') }}</el-radio>
            <el-radio label="once">{{ $t('cronTask.onceExec') }}</el-radio>
          </el-radio-group>
        </el-form-item>

        <!-- Cron表达式 -->
        <el-form-item v-if="form.scheduleType === 'cron'" :label="$t('cronTask.cronExpression')" prop="cronSpec">
          <el-input v-model="form.cronSpec" :placeholder="$t('cronTask.cronPlaceholder')">
            <template #append>
              <el-button @click="validateCron">{{ $t('cronTask.validate') }}</el-button>
            </template>
          </el-input>
          <div class="cron-help">
            <div class="cron-presets">
              <span class="preset-label">{{ $t('cronTask.quickSelect') }}</span>
              <el-tag 
                v-for="preset in cronPresets" 
                :key="preset.value" 
                size="small" 
                class="preset-tag"
                @click="form.cronSpec = preset.value; validateCron()"
              >
                {{ preset.label }}
              </el-tag>
            </div>
            <div v-if="cronValidation.valid" class="cron-next-times">
              <div class="next-label">{{ $t('cronTask.next5Times') }}</div>
              <div v-for="(time, index) in cronValidation.nextTimes" :key="index" class="next-time">
                {{ index + 1 }}. {{ time }}
              </div>
            </div>
            <div v-else-if="cronValidation.error" class="cron-error">
              {{ cronValidation.error }}
            </div>
          </div>
        </el-form-item>

        <!-- 指定时间 -->
        <el-form-item v-if="form.scheduleType === 'once'" :label="$t('cronTask.execTime')" prop="scheduleTime">
          <el-date-picker
            v-model="form.scheduleTimeDate"
            type="datetime"
            :placeholder="$t('common.pleaseSelect')"
            format="YYYY-MM-DD HH:mm:ss"
            value-format="YYYY-MM-DD HH:mm:ss"
            :disabled-date="disabledDate"
            style="width: 100%"
            @change="onScheduleTimeChange"
          />
          <div class="form-hint">{{ $t('cronTask.onceExecHint') }}</div>
        </el-form-item>

        <!-- 新建时提示：复用关联任务配置 -->
        <el-alert
          v-if="!isEdit && form.mainTaskId"
          :title="$t('cronTask.reuseConfigHint') || '将复用关联任务的扫描配置，创建后可在编辑中调整'"
          type="info"
          :closable="false"
          show-icon
          style="margin-bottom: 15px"
        />

        <!-- 扫描配置折叠面板 - 仅编辑时显示 -->
        <el-collapse v-if="isEdit" v-model="activeCollapse" class="config-collapse">
          <!-- 子域名扫描 -->
          <el-collapse-item name="domainscan">
            <template #title>
              <span class="collapse-title">{{ $t('task.subdomainScan') }} <el-tag v-if="form.domainscanEnable" type="success" size="small">{{ $t('task.started') }}</el-tag></span>
            </template>
            <el-form-item :label="$t('task.enable')">
              <el-switch v-model="form.domainscanEnable" />
              <span class="form-hint">{{ $t('task.subdomainEnumHint') }}</span>
            </el-form-item>
            <template v-if="form.domainscanEnable">
              <el-form-item :label="$t('task.scanTool')">
                <el-checkbox v-model="form.domainscanSubfinder">Subfinder ({{ $t('task.passiveEnum') }})</el-checkbox>
                <el-checkbox v-model="form.domainscanBruteforce" :disabled="!form.subdomainDictIds || !form.subdomainDictIds.length">KSubdomain ({{ $t('task.dictBrute') }})</el-checkbox>
                <span class="form-hint">{{ $t('task.multiScanHint') }}</span>
              </el-form-item>
              
              <!-- 左右分栏布局 -->
              <el-row :gutter="24" class="scan-tools-layout">
                <!-- 左侧：Subfinder 配置 -->
                <el-col :span="12">
                  <div class="scan-tool-section">
                    <div class="scan-tool-header">
                      <span class="scan-tool-title">{{ $t('task.subfinderPassiveEnum') }}</span>
                      <el-tag :type="form.domainscanSubfinder ? 'success' : 'info'" size="small">
                        {{ form.domainscanSubfinder ? $t('task.started') : $t('task.notStarted') }}
                      </el-tag>
                    </div>
                    <template v-if="form.domainscanSubfinder">
                      <el-form-item :label="$t('task.timeoutSeconds')">
                        <el-input-number v-model="form.domainscanTimeout" :min="60" :max="3600" style="width:100%" />
                      </el-form-item>
                      <el-form-item :label="$t('task.maxEnumTime') + '(' + $t('task.minutes') + ')'">
                        <el-input-number v-model="form.domainscanMaxEnumTime" :min="1" :max="60" style="width:100%" />
                      </el-form-item>
                      <el-form-item :label="$t('task.rateLimit')">
                        <el-input-number v-model="form.domainscanRateLimit" :min="0" :max="1000" style="width:100%" />
                        <span class="form-hint">0={{ $t('task.noLimit') }}</span>
                      </el-form-item>
                      <el-form-item :label="$t('task.scanOptions')">
                        <el-checkbox v-model="form.domainscanRemoveWildcard">{{ $t('task.removeWildcardDomain') }}</el-checkbox>
                      </el-form-item>
                      <el-form-item :label="$t('task.dnsResolve')">
                        <el-checkbox v-model="form.domainscanResolveDNS">{{ $t('task.resolveSubdomainDns') }}</el-checkbox>
                        <span class="form-hint">{{ $t('task.concurrentByWorker') }}</span>
                      </el-form-item>
                    </template>
                    <div v-else class="scan-tool-disabled-hint">
                      <el-icon><InfoFilled /></el-icon>
                      <span>{{ $t('task.enableSubfinderFirst') }}</span>
                    </div>
                  </div>
                </el-col>
                
                <!-- 右侧：KSubdomain 配置 -->
                <el-col :span="12">
                  <div class="scan-tool-section">
                    <div class="scan-tool-header">
                      <span class="scan-tool-title">{{ $t('task.ksubdomainDictBrute') }}</span>
                      <el-tag :type="form.domainscanBruteforce ? 'success' : 'info'" size="small">
                        {{ form.domainscanBruteforce ? $t('task.started') : $t('task.notStarted') }}
                      </el-tag>
                    </div>
                    <!-- 字典选择（始终显示，作为启用字典爆破的前提） -->
                    <el-form-item :label="$t('task.bruteforceDict')">
                      <div class="selected-dict-summary">
                        <el-tag type="primary" size="small" v-if="form.subdomainDictIds && form.subdomainDictIds.length">
                          {{ $t('task.selectedCount', { count: form.subdomainDictIds.length }) }}
                        </el-tag>
                        <span v-else class="warning-hint">
                          {{ $t('task.selectDictFirst') }}
                        </span>
                        <el-button type="primary" link @click="showSubdomainDictSelectDialog">{{ $t('task.selectDict') }}</el-button>
                      </div>
                      <span class="form-hint">{{ $t('task.ksubdomainBruteHint') }}</span>
                    </el-form-item>
                    <template v-if="form.domainscanBruteforce">
                      <el-form-item :label="$t('task.bruteforceTimeout') + ' (' + $t('task.minutes') + ')'">
                        <el-input-number v-model="form.domainscanBruteforceTimeout" :min="1" :max="120" style="width:100%" />
                        <span class="form-hint">{{ $t('task.ksubdomainTimeoutHint') }}</span>
                      </el-form-item>
                    </template>
                    <template v-if="form.domainscanBruteforce">
                      <el-form-item :label="$t('task.enhancedFeatures')">
                        <div style="display: flex; flex-direction: column; gap: 8px;">
                          <div style="display: flex; align-items: center; gap: 8px;">
                            <el-checkbox 
                              v-model="form.domainscanRecursiveBrute" 
                              :disabled="!form.recursiveDictIds || !form.recursiveDictIds.length"
                            >{{ $t('task.recursiveBrute') }}</el-checkbox>
                            <el-button type="primary" link size="small" @click="showRecursiveDictSelectDialog">{{ $t('task.selectRecursiveDict') }}</el-button>
                            <el-tag type="primary" size="small" v-if="form.recursiveDictIds && form.recursiveDictIds.length">
                              {{ $t('task.selectedCount', { count: form.recursiveDictIds.length }) }}
                            </el-tag>
                          </div>
                          <span class="form-hint" style="margin-left: 24px; margin-top: -4px;">
                            {{ (!form.recursiveDictIds || !form.recursiveDictIds.length) ? $t('task.selectRecursiveDictFirst') : $t('task.recursiveBruteHint') }}
                          </span>
                          <el-checkbox v-model="form.domainscanWildcardDetect">{{ $t('task.wildcardDetect') }}</el-checkbox>
                          <span class="form-hint" style="margin-left: 24px; margin-top: -4px;">{{ $t('task.wildcardDetectHint') }}</span>
                          
                          
                        </div>
                      </el-form-item>
                    </template>
                    <div v-if="!form.domainscanBruteforce && form.subdomainDictIds && form.subdomainDictIds.length" class="scan-tool-disabled-hint">
                      <el-icon><InfoFilled /></el-icon>
                      <span>{{ $t('task.canEnableKSubdomain') }}</span>
                    </div>
                  </div>
                </el-col>
              </el-row>
            </template>
          </el-collapse-item>

          <!-- 端口扫描 -->
          <el-collapse-item name="portscan">
            <template #title>
              <span class="collapse-title">{{ $t('task.portScan') }} <el-tag v-if="form.portscanEnable" type="success" size="small">{{ $t('task.started') }}</el-tag></span>
            </template>
            <el-form-item :label="$t('task.enable')">
              <el-switch v-model="form.portscanEnable" />
            </el-form-item>
            <template v-if="form.portscanEnable">
              <el-form-item :label="$t('task.scanTool')">
                <el-radio-group v-model="form.portscanTool">
                  <el-radio label="naabu">Naabu ({{ $t('task.recommended') }})</el-radio>
                  <el-radio label="masscan">Masscan</el-radio>
                </el-radio-group>
              </el-form-item>
              <el-form-item :label="$t('task.portRange')">
                <el-select v-model="form.ports" filterable allow-create default-first-option style="width: 100%">
                  <el-option :label="$t('task.top100Ports')" value="top100" />
                  <el-option :label="$t('task.top1000Ports')" value="top1000" />
                  <el-option :label="'80,443,8080,8443 - ' + $t('task.webCommon')" value="80,443,8080,8443" />
                  <el-option :label="'1-65535 - ' + $t('task.allPorts')" value="1-65535" />
                </el-select>
              </el-form-item>
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item :label="$t('task.scanRate')">
                    <el-input-number v-model="form.portscanRate" :min="100" :max="100000" style="width:100%" />
                    <span class="form-hint">{{ $t('task.packetsPerSecond') }}</span>
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item :label="$t('task.portThreshold')">
                    <el-input-number v-model="form.portThreshold" :min="0" :max="65535" style="width:100%" />
                    <span class="form-hint">{{ $t('task.skipIfExceeded') }}</span>
                  </el-form-item>
                </el-col>
              </el-row>
              <el-row :gutter="20" v-if="form.portscanTool === 'naabu'">
                <el-col :span="12">
                  <el-form-item :label="$t('task.workers')">
                    <el-input-number v-model="form.portscanWorkers" :min="10" :max="200" style="width:100%" />
                    <span class="form-hint">{{ $t('task.internalThreads') }}</span>
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item :label="$t('task.retries')">
                    <el-input-number v-model="form.portscanRetries" :min="0" :max="5" style="width:100%" />
                    <span class="form-hint">{{ $t('task.retryCount') }}</span>
                  </el-form-item>
                </el-col>
              </el-row>
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item v-if="form.portscanTool === 'naabu'" :label="$t('task.scanType')">
                    <el-radio-group v-model="form.scanType">
                      <el-radio label="c">CONNECT</el-radio>
                      <el-radio label="s">SYN</el-radio>
                    </el-radio-group>
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item :label="$t('task.timeoutSeconds')">
                    <el-input-number v-model="form.portscanTimeout" :min="5" :max="1200" style="width:100%" />
                  </el-form-item>
                </el-col>
              </el-row>
              <el-row :gutter="20" v-if="form.portscanTool === 'naabu'">
                <el-col :span="12">
                  <el-form-item :label="$t('task.warmUpTime')">
                    <el-input-number v-model="form.portscanWarmUpTime" :min="0" :max="10" style="width:100%" />
                    <span class="form-hint">{{ $t('task.warmUpTimeHint') }}</span>
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item :label="$t('task.tcpVerify')">
                    <el-switch v-model="form.portscanVerify" />
                    <span class="form-hint">{{ $t('task.tcpVerifyHint') }}</span>
                  </el-form-item>
                </el-col>
              </el-row>
              <el-form-item :label="$t('task.advancedOptions')">
                <div style="display: block; width: 100%">
                  <el-checkbox v-model="form.skipHostDiscovery">{{ $t('task.skipHostDiscovery') }} (-Pn)</el-checkbox>
                  <span class="form-hint">{{ $t('task.skipHostDiscoveryHint') }}</span>
                </div>
                <div v-if="form.portscanTool === 'naabu'" style="display: block; width: 100%; margin-top: 8px">
                  <el-checkbox v-model="form.excludeCDN">{{ $t('task.excludeCdnWaf') }} (-ec)</el-checkbox>
                  <span class="form-hint">{{ $t('task.excludeCdnHint') }}</span>
                </div>
              </el-form-item>
              <el-form-item :label="$t('task.excludeTargets')">
                <el-input v-model="form.excludeHosts" placeholder="192.168.1.1,10.0.0.0/8" />
                <span class="form-hint">{{ $t('task.excludeTargetsHint') }}</span>
              </el-form-item>
            </template>
          </el-collapse-item>

          <!-- 端口识别 -->
          <el-collapse-item name="portidentify">
            <template #title>
              <span class="collapse-title">{{ $t('task.portIdentify') }} <el-tag v-if="form.portidentifyEnable" type="success" size="small">{{ $t('task.started') }}</el-tag></span>
            </template>
            <el-form-item :label="$t('task.enable')">
              <el-switch v-model="form.portidentifyEnable" />
            </el-form-item>
            <template v-if="form.portidentifyEnable">
              <!-- 强制扫描：仅在端口扫描未启用时显示 -->
              <el-form-item v-if="!form.portscanEnable" :label="$t('task.forceScan')">
                <el-switch v-model="form.portidentifyForceScan" />
                <span class="form-hint warning-hint">{{ $t('task.forceScanHint') }}</span>
              </el-form-item>
              <el-form-item :label="$t('task.identifyTool')">
                <el-radio-group v-model="form.portidentifyTool">
                  <el-radio label="nmap">Nmap</el-radio>
                  <el-radio label="fingerprintx">Fingerprintx</el-radio>
                </el-radio-group>
              </el-form-item>
              <el-form-item :label="$t('task.timeoutSeconds')">
                <el-input-number v-model="form.portidentifyTimeout" :min="5" :max="300" />
                <span class="form-hint">{{ $t('task.singleHostTimeout') }}</span>
              </el-form-item>
              <el-form-item v-if="form.portidentifyTool === 'fingerprintx'" :label="$t('task.concurrent')">
                <el-input-number v-model="form.portidentifyConcurrency" :min="1" :max="100" />
              </el-form-item>
              <el-form-item v-if="form.portidentifyTool === 'nmap'" :label="$t('task.nmapParams')">
                <el-input v-model="form.portidentifyArgs" placeholder="-sV --version-intensity 5" />
              </el-form-item>
              <el-form-item v-if="form.portidentifyTool === 'fingerprintx'" :label="$t('task.scanUDP')">
                <el-switch v-model="form.portidentifyUDP" />
              </el-form-item>
              <el-form-item v-if="form.portidentifyTool === 'fingerprintx'" :label="$t('task.fastMode')">
                <el-switch v-model="form.portidentifyFastMode" />
              </el-form-item>
            </template>
          </el-collapse-item>

          <!-- 指纹识别 -->
          <el-collapse-item name="fingerprint">
            <template #title>
              <span class="collapse-title">{{ $t('task.fingerprintScan') }} <el-tag v-if="form.fingerprintEnable" type="success" size="small">{{ $t('task.started') }}</el-tag></span>
            </template>
            <el-form-item :label="$t('task.enable')">
              <el-switch v-model="form.fingerprintEnable" />
            </el-form-item>
            <template v-if="form.fingerprintEnable">
              <!-- 强制扫描：仅在端口扫描和端口识别均未启用时显示 -->
              <el-form-item v-if="!form.portscanEnable && !form.portidentifyEnable" :label="$t('task.forceScan')">
                <el-switch v-model="form.fingerprintForceScan" />
                <span class="form-hint warning-hint">{{ $t('task.forceScanHint') }}</span>
              </el-form-item>
              <el-form-item :label="$t('task.probeTool')">
                <el-radio-group v-model="form.fingerprintTool">
                  <el-radio label="httpx">Httpx</el-radio>
                  <el-radio label="builtin">{{ $t('task.builtinEngine') }}</el-radio>
                </el-radio-group>
                <span class="form-hint">{{ form.fingerprintTool === 'httpx' ? $t('task.httpxWappalyzer') : $t('task.sdkWappalyzer') }}</span>
              </el-form-item>
              <el-form-item :label="$t('task.additionalFeatures')">
                <el-checkbox v-model="form.fingerprintIconHash">{{ $t('task.iconHash') }}</el-checkbox>
                <el-checkbox v-model="form.fingerprintCustomEngine">{{ $t('task.customFingerprint') }}</el-checkbox>
                <el-checkbox v-model="form.fingerprintScreenshot">{{ $t('task.screenshot') }}</el-checkbox>
              </el-form-item>
              <el-form-item :label="$t('task.filterMode')">
                <el-radio-group v-model="form.fingerprintFilterMode">
                  <el-radio label="http_mapping">{{ $t('task.httpMappingMode') }}</el-radio>
                  <el-radio label="service_mapping">{{ $t('task.serviceMappingMode') }}</el-radio>
                </el-radio-group>
                <span class="form-hint">{{ form.fingerprintFilterMode === 'http_mapping' ? $t('task.httpMappingModeHint') : $t('task.serviceMappingModeHint') }}</span>
              </el-form-item>
              <el-form-item :label="$t('task.activeScan')">
                <el-checkbox v-model="form.fingerprintActiveScan">{{ $t('task.enableActiveScan') }}</el-checkbox>
                <span class="form-hint">{{ $t('task.activeScanHint') }}</span>
              </el-form-item>
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item :label="$t('task.timeoutSeconds')">
                    <el-input-number v-model="form.fingerprintTimeout" :min="5" :max="120" style="width:100%" />
                    <span class="form-hint">{{ $t('task.concurrentByWorker') }}</span>
                  </el-form-item>
                </el-col>
                <el-col :span="12" v-if="form.fingerprintActiveScan">
                  <el-form-item :label="$t('task.activeTimeoutSeconds')">
                    <el-input-number v-model="form.fingerprintActiveTimeout" :min="5" :max="60" style="width:100%" />
                    <span class="form-hint">{{ $t('task.activeProbeTimeout') }}</span>
                  </el-form-item>
                </el-col>
              </el-row>
            </template>
          </el-collapse-item>

          <!-- 目录扫描 -->
          <el-collapse-item name="dirscan">
            <template #title>
              <span class="collapse-title">{{ $t('task.dirScan') }} <el-tag v-if="form.dirscanEnable" type="success" size="small">{{ $t('task.started') }}</el-tag></span>
            </template>
            <el-form-item :label="$t('task.enable')">
              <el-switch v-model="form.dirscanEnable" />
              <span class="form-hint">{{ $t('task.dirScanHint') }}</span>
            </el-form-item>
            <template v-if="form.dirscanEnable">
              <!-- 强制扫描：仅在前序阶段均未启用时显示 -->
              <el-form-item v-if="!hasPrePhaseEnabled" :label="$t('task.forceScan')">
                <el-switch v-model="form.dirscanForceScan" />
                <span class="form-hint warning-hint">{{ $t('task.forceScanHint') }}</span>
              </el-form-item>
              <el-form-item :label="$t('task.scanDict')">
                <div class="selected-dict-summary">
                  <el-tag type="primary" size="small" v-if="form.dirscanDictIds.length">
                    {{ $t('task.selectedCount', { count: form.dirscanDictIds.length }) }}
                  </el-tag>
                  <span v-if="!form.dirscanDictIds.length" class="secondary-hint">
                    {{ $t('task.noDictSelected') }}
                  </span>
                  <el-button type="primary" link @click="showDictSelectDialog">{{ $t('task.selectDict') }}</el-button>
                </div>
              </el-form-item>
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item :label="$t('task.concurrentThreads')">
                    <el-input-number v-model="form.dirscanThreads" :min="1" :max="200" style="width:100%" />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item :label="$t('task.requestTimeoutSeconds')">
                    <el-input-number v-model="form.dirscanTimeout" :min="1" :max="60" style="width:100%" />
                  </el-form-item>
                </el-col>
              </el-row>
              <el-form-item :label="$t('task.followRedirect')">
                <el-switch v-model="form.dirscanFollowRedirect" />
              </el-form-item>
              <!-- ffuf 高级配置 -->
              <el-divider content-position="left">{{ $t('task.ffufAdvanced') }}</el-divider>
              <el-form-item :label="$t('task.autoCalibration')">
                <el-switch v-model="form.dirscanAutoCalibration" />
                <span class="form-hint">{{ $t('task.autoCalibrationHint') }}</span>
              </el-form-item>
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item :label="$t('task.rateLimit')">
                    <el-input-number v-model="form.dirscanRate" :min="0" :max="10000" style="width:100%" />
                    <span class="form-hint">{{ $t('task.rateLimitHint') }}</span>
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item :label="$t('task.recursion')">
                    <el-switch v-model="form.dirscanRecursion" />
                  </el-form-item>
                </el-col>
              </el-row>
              <el-row :gutter="20" v-if="form.dirscanRecursion">
                <el-col :span="12">
                  <el-form-item :label="$t('task.recursionDepth')">
                    <el-input-number v-model="form.dirscanRecursionDepth" :min="1" :max="10" style="width:100%" />
                  </el-form-item>
                </el-col>
              </el-row>
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item :label="$t('task.filterMode')">
                    <el-select v-model="form.dirscanFilterMode" style="width:100%">
                      <el-option label="OR" value="or" />
                      <el-option label="AND" value="and" />
                    </el-select>
                  </el-form-item>
                </el-col>
              </el-row>
              <!-- 过滤条件组 -->
              <div class="filter-group">
                <div class="filter-group-title">{{ $t('task.filterConditions') }}</div>
                <el-row :gutter="20">
                  <el-col :span="6">
                    <el-form-item :label="$t('task.filterSize')">
                      <el-input v-model="form.dirscanFilterSize" :placeholder="$t('task.filterSizeHint')" />
                    </el-form-item>
                  </el-col>
                  <el-col :span="6">
                    <el-form-item :label="$t('task.filterWords')">
                      <el-input v-model="form.dirscanFilterWords" :placeholder="$t('task.filterWordsHint')" />
                    </el-form-item>
                  </el-col>
                  <el-col :span="6">
                    <el-form-item :label="$t('task.filterLines')">
                      <el-input v-model="form.dirscanFilterLines" :placeholder="$t('task.filterLinesHint')" />
                    </el-form-item>
                  </el-col>
                  <el-col :span="6">
                    <el-form-item :label="$t('task.filterRegex')">
                      <el-input v-model="form.dirscanFilterRegex" :placeholder="$t('task.filterRegexHint')" />
                    </el-form-item>
                  </el-col>
                </el-row>
              </div>
            </template>
          </el-collapse-item>

          <!-- 漏洞扫描 -->
          <el-collapse-item name="pocscan">
            <template #title>
              <span class="collapse-title">{{ $t('task.vulScan') }} <el-tag v-if="form.pocscanEnable" type="success" size="small">{{ $t('task.started') }}</el-tag></span>
            </template>
            <el-form-item :label="$t('task.enable')">
              <el-switch v-model="form.pocscanEnable" />
              <span class="form-hint">{{ $t('task.useNucleiEngine') }}</span>
            </el-form-item>
            <template v-if="form.pocscanEnable">
              <!-- 强制扫描：仅在前序阶段均未启用时显示 -->
              <el-form-item v-if="!hasPrePhaseEnabled" :label="$t('task.forceScan')">
                <el-switch v-model="form.pocscanForceScan" />
                <span class="form-hint warning-hint">{{ $t('task.forceScanHint') }}</span>
              </el-form-item>
              <el-form-item :label="$t('task.pocSource')">
                <el-radio-group v-model="form.pocscanMode" @change="handlePocModeChange">
                  <el-radio label="auto">{{ $t('task.autoMatch') }}</el-radio>
                  <el-radio label="manual">{{ $t('task.manualSelect') }}</el-radio>
                </el-radio-group>
              </el-form-item>
              
              <!-- 自动匹配模式 -->
              <template v-if="form.pocscanMode === 'auto'">
                <el-form-item :label="$t('task.autoScan')">
                  <el-checkbox v-model="form.pocscanAutoScan" :disabled="form.pocscanCustomOnly">{{ $t('task.customTagMapping') }}</el-checkbox>
                  <el-checkbox v-model="form.pocscanAutomaticScan" :disabled="form.pocscanCustomOnly || !form.fingerprintEnable">{{ $t('task.webFingerprintAutoMatch') }}</el-checkbox>
                  <span v-if="!form.fingerprintEnable && !form.pocscanCustomOnly" class="form-hint warning-hint">{{ $t('task.needFingerprintScan') }}</span>
                </el-form-item>
                <el-form-item :label="$t('task.customPoc')">
                  <el-checkbox v-model="form.pocscanCustomOnly">{{ $t('task.onlyUseCustomPoc') }}</el-checkbox>
                </el-form-item>
              </template>
              
              <!-- 手动选择模式 -->
              <template v-if="form.pocscanMode === 'manual'">
                <el-form-item :label="$t('task.selectedPoc')">
                  <div class="selected-poc-summary">
                    <el-tag type="primary" size="small" v-if="form.pocscanNucleiTemplateIds.length">
                      {{ $t('task.defaultTemplate') }}: {{ form.pocscanNucleiTemplateIds.length }}
                    </el-tag>
                    <el-tag type="warning" size="small" v-if="form.pocscanCustomPocIds.length">
                      {{ $t('task.customPoc') }}: {{ form.pocscanCustomPocIds.length }}
                    </el-tag>
                    <span v-if="!form.pocscanNucleiTemplateIds.length && !form.pocscanCustomPocIds.length" class="secondary-hint">
                      {{ $t('task.noPocSelected') }}
                    </span>
                    <el-button type="primary" link @click="showPocSelectDialog">{{ $t('task.selectPoc') }}</el-button>
                  </div>
                </el-form-item>
              </template>

              <el-form-item v-if="form.pocscanMode !== 'manual'" :label="$t('task.severityLevel')">
                <el-checkbox-group v-model="form.pocscanSeverity">
                  <el-checkbox label="critical">Critical</el-checkbox>
                  <el-checkbox label="high">High</el-checkbox>
                  <el-checkbox label="medium">Medium</el-checkbox>
                  <el-checkbox label="low">Low</el-checkbox>
                  <el-checkbox label="info">Info</el-checkbox>
                  <el-checkbox label="unknown">Unknown</el-checkbox>
                </el-checkbox-group>
              </el-form-item>
              <el-form-item label="请求速率(Rate/s)">
                <el-input-number v-model="form.pocscanRateLimit" :min="1" :max="2000" />
              </el-form-item>
              <el-form-item label="模板并发">
                <el-input-number v-model="form.pocscanConcurrency" :min="1" :max="500" />
              </el-form-item>
              <el-form-item :label="$t('task.targetTimeout')">
                <el-input-number v-model="form.pocscanTargetTimeout" :min="30" :max="600" />
                <span class="form-hint">{{ $t('task.seconds') }}</span>
              </el-form-item>
              <el-form-item :label="$t('task.customHeaders')">
                <el-radio-group v-model="form.pocscanHeaderMode" style="margin-bottom: 8px;">
                  <el-radio label="none">{{ $t('task.noCustomHeader') }}</el-radio>
                  <el-radio label="preset">{{ $t('task.presetUA') }}</el-radio>
                  <el-radio label="custom">{{ $t('task.customInput') }}</el-radio>
                </el-radio-group>
                <template v-if="form.pocscanHeaderMode === 'preset'">
                  <el-select v-model="form.pocscanPresetUA" :placeholder="$t('task.selectUA')" style="width: 100%;">
                    <el-option-group :label="$t('task.uaDesktop')">
                      <el-option label="Chrome (Windows)" value="Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36" />
                      <el-option label="Firefox (macOS)" value="Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:123.0) Gecko/20100101 Firefox/123.0" />
                      <el-option label="Edge (Windows)" value="Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 Edg/122.0.0.0" />
                    </el-option-group>
                    <el-option-group :label="$t('task.uaMobile')">
                      <el-option label="Safari (iPhone)" value="Mozilla/5.0 (iPhone; CPU iPhone OS 17_3_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3.1 Mobile/15E148 Safari/604.1" />
                      <el-option label="Chrome (Android)" value="Mozilla/5.0 (Linux; Android 13; SM-S918B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Mobile Safari/537.36" />
                    </el-option-group>
                    <el-option-group :label="$t('task.uaSpider')">
                      <el-option label="Baiduspider" value="Mozilla/5.0 (compatible; Baiduspider/2.0; +http://www.baidu.com/search/spider.html)" />
                      <el-option label="Googlebot" value="Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)" />
                    </el-option-group>
                    <el-option-group :label="$t('task.uaApp')">
                      <el-option label="WeChat (Android)" value="Mozilla/5.0 (Linux; Android 13; ALN-AL00 Build/HUAWEIALN-AL00; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/116.0.0.0 Mobile Safari/537.36 XWEB/1160065 MMWEBSDK/20231202 MicroMessenger/8.0.47.2560 WeChat/arm64 Weixin NetType/WIFI" />
                    </el-option-group>
                  </el-select>
                </template>
                <template v-if="form.pocscanHeaderMode === 'custom'">
                  <el-input
                    v-model="form.pocscanCustomHeadersText"
                    type="textarea"
                    :rows="4"
                    :placeholder="$t('task.customHeadersPlaceholder')"
                  />
                </template>
              </el-form-item>
            </template>
          </el-collapse-item>

          <!-- 高级设置 -->
          <el-collapse-item name="advanced">
            <template #title>
              <span class="collapse-title">{{ $t('task.advancedSettings') }}</span>
            </template>
            <el-form-item :label="$t('task.taskSplit')">
              <el-input-number v-model="form.batchSize" :min="0" :max="1000" :step="10" />
              <span class="form-hint">{{ $t('task.batchTargetCount') }}</span>
            </el-form-item>
          </el-collapse-item>
        </el-collapse>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">{{ $t('common.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- 目录扫描字典选择对话框 -->
    <el-dialog v-model="dictSelectDialogVisible" :title="$t('task.selectDirScanDict')" width="800px" @open="handleDictDialogOpen">
      <el-table 
        ref="dictTableRef"
        :data="dictList" 
        v-loading="dictLoading" 
        max-height="400"
        @selection-change="handleDictSelectionChange"
        row-key="id"
      >
        <el-table-column type="selection" width="45" :reserve-selection="true" />
        <el-table-column prop="name" :label="$t('task.dictName')" min-width="150" />
        <el-table-column prop="pathCount" :label="$t('task.pathCount')" width="100" />
        <el-table-column prop="isBuiltin" :label="$t('common.type')" width="80">
          <template #default="{ row }">
            <el-tag :type="row.isBuiltin ? 'info' : 'success'" size="small">{{ row.isBuiltin ? $t('task.builtin') : $t('task.custom') }}</el-tag>
          </template>
        </el-table-column>
      </el-table>
      <template #footer>
        <el-button @click="dictSelectDialogVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="confirmDictSelection">{{ $t('common.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- 子域名字典选择对话框 -->
    <el-dialog v-model="subdomainDictSelectDialogVisible" :title="$t('task.selectSubdomainDict')" width="800px" @open="handleSubdomainDictDialogOpen">
      <el-table 
        ref="subdomainDictTableRef"
        :data="subdomainDictList" 
        v-loading="subdomainDictLoading" 
        max-height="400"
        @selection-change="handleSubdomainDictSelectionChange"
        row-key="id"
      >
        <el-table-column type="selection" width="45" :reserve-selection="true" />
        <el-table-column prop="name" :label="$t('task.dictName')" min-width="150" />
        <el-table-column prop="wordCount" :label="$t('task.wordCount')" width="100" />
        <el-table-column prop="isBuiltin" :label="$t('common.type')" width="80">
          <template #default="{ row }">
            <el-tag :type="row.isBuiltin ? 'info' : 'success'" size="small">{{ row.isBuiltin ? $t('task.builtin') : $t('task.custom') }}</el-tag>
          </template>
        </el-table-column>
      </el-table>
      <template #footer>
        <el-button @click="subdomainDictSelectDialogVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="confirmSubdomainDictSelection">{{ $t('common.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- 递归爆破字典选择对话框 -->
    <el-dialog v-model="recursiveDictSelectDialogVisible" :title="$t('task.selectRecursiveDict')" width="800px" @open="handleRecursiveDictDialogOpen">
      <el-table 
        ref="recursiveDictTableRef"
        :data="recursiveDictList" 
        v-loading="recursiveDictLoading" 
        max-height="400"
        @selection-change="handleRecursiveDictSelectionChange"
        row-key="id"
      >
        <el-table-column type="selection" width="45" :reserve-selection="true" />
        <el-table-column prop="name" :label="$t('task.dictName')" min-width="150" />
        <el-table-column prop="wordCount" :label="$t('task.wordCount')" width="100" />
        <el-table-column prop="isBuiltin" :label="$t('common.type')" width="80">
          <template #default="{ row }">
            <el-tag :type="row.isBuiltin ? 'info' : 'success'" size="small">{{ row.isBuiltin ? $t('task.builtin') : $t('task.custom') }}</el-tag>
          </template>
        </el-table-column>
      </el-table>
      <template #footer>
        <el-button @click="recursiveDictSelectDialogVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="confirmRecursiveDictSelection">{{ $t('common.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- POC选择对话框 -->
    <el-dialog v-model="pocSelectDialogVisible" :title="$t('task.selectPoc')" width="1200px" @open="handlePocDialogOpen">
      <div class="poc-select-container">
        <!-- 左侧：POC列表 -->
        <div class="poc-select-left">
          <el-tabs v-model="pocSelectTab">
            <!-- 默认模板 -->
            <el-tab-pane :label="$t('task.defaultTemplate')" name="nuclei">
              <el-form :inline="true" class="poc-filter-form">
                <el-form-item>
                  <el-input v-model="nucleiTemplateFilter.keyword" :placeholder="$t('task.nameOrId')" clearable style="width: 150px" @keyup.enter="loadNucleiTemplatesForSelect" />
                </el-form-item>
                <el-form-item>
                  <el-select v-model="nucleiTemplateFilter.severity" :placeholder="$t('task.level')" clearable style="width: 100px" @change="loadNucleiTemplatesForSelect">
                    <el-option label="Critical" value="critical" />
                    <el-option label="High" value="high" />
                    <el-option label="Medium" value="medium" />
                    <el-option label="Low" value="low" />
                    <el-option label="Info" value="info" />
                  </el-select>
                </el-form-item>
                <el-form-item>
                  <el-button type="primary" size="small" @click="loadNucleiTemplatesForSelect">{{ $t('common.search') }}</el-button>
                  <el-button type="success" size="small" @click="selectAllNucleiTemplates" :loading="selectAllNucleiLoading">{{ $t('task.selectAll') }}</el-button>
                  <el-button type="warning" size="small" @click="deselectAllNucleiTemplates" v-if="selectedNucleiTemplateIds.length > 0">{{ $t('task.deselectAll') }}</el-button>
                </el-form-item>
              </el-form>
              <el-table 
                ref="nucleiTableRef"
                :data="nucleiTemplateList" 
                v-loading="nucleiTemplateLoading" 
                max-height="400"
                @selection-change="handleNucleiSelectionChange"
                row-key="id"
              >
                <el-table-column type="selection" width="45" :reserve-selection="true" />
                <el-table-column prop="id" :label="$t('task.templateId')" width="180" show-overflow-tooltip />
                <el-table-column prop="name" :label="$t('common.name')" min-width="150" show-overflow-tooltip />
                <el-table-column prop="severity" :label="$t('task.level')" width="80">
                  <template #default="{ row }">
                    <el-tag :type="getSeverityType(row.severity)" size="small">{{ row.severity }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column :label="$t('common.operation')" width="60" fixed="right">
                  <template #default="{ row }">
                    <el-button type="primary" link size="small" @click="viewPocContent(row, 'nuclei')">{{ $t('common.view') }}</el-button>
                  </template>
                </el-table-column>
              </el-table>
              <el-pagination
                v-model:current-page="nucleiTemplatePagination.page"
                v-model:page-size="nucleiTemplatePagination.pageSize"
                :total="nucleiTemplatePagination.total"
                :page-sizes="[50, 100, 200]"
                layout="total, sizes, prev, pager, next"
                class="poc-pagination"
                @size-change="loadNucleiTemplatesForSelect"
                @current-change="loadNucleiTemplatesForSelect"
              />
            </el-tab-pane>

            <!-- 自定义POC -->
            <el-tab-pane :label="$t('task.customPoc')" name="custom">
              <el-form :inline="true" class="poc-filter-form">
                <el-form-item>
                  <el-input v-model="customPocFilter.name" :placeholder="$t('common.name')" clearable style="width: 150px" @keyup.enter="loadCustomPocsForSelect" />
                </el-form-item>
                <el-form-item>
                  <el-select v-model="customPocFilter.severity" :placeholder="$t('task.level')" clearable style="width: 100px" @change="loadCustomPocsForSelect">
                    <el-option label="Critical" value="critical" />
                    <el-option label="High" value="high" />
                    <el-option label="Medium" value="medium" />
                    <el-option label="Low" value="low" />
                    <el-option label="Info" value="info" />
                  </el-select>
                </el-form-item>
                <el-form-item>
                  <el-button type="primary" size="small" @click="loadCustomPocsForSelect">{{ $t('common.search') }}</el-button>
                  <el-button type="success" size="small" @click="selectAllCustomPocs" :loading="selectAllCustomLoading">{{ $t('task.selectAll') }}</el-button>
                  <el-button type="warning" size="small" @click="deselectAllCustomPocs" v-if="selectedCustomPocIds.length > 0">{{ $t('task.deselectAll') }}</el-button>
                </el-form-item>
              </el-form>
              <el-table 
                ref="customPocTableRef"
                :data="customPocList" 
                v-loading="customPocLoading" 
                max-height="400"
                @selection-change="handleCustomPocSelectionChange"
                row-key="id"
              >
                <el-table-column type="selection" width="45" :reserve-selection="true" />
                <el-table-column prop="name" :label="$t('common.name')" min-width="150" show-overflow-tooltip />
                <el-table-column prop="templateId" :label="$t('task.templateId')" width="150" show-overflow-tooltip />
                <el-table-column prop="severity" :label="$t('task.level')" width="80">
                  <template #default="{ row }">
                    <el-tag :type="getSeverityType(row.severity)" size="small">{{ row.severity }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column :label="$t('common.operation')" width="60" fixed="right">
                  <template #default="{ row }">
                    <el-button type="primary" link size="small" @click="viewPocContent(row, 'custom')">{{ $t('common.view') }}</el-button>
                  </template>
                </el-table-column>
              </el-table>
              <el-pagination
                v-model:current-page="customPocPagination.page"
                v-model:page-size="customPocPagination.pageSize"
                :total="customPocPagination.total"
                :page-sizes="[50, 100, 200]"
                layout="total, sizes, prev, pager, next"
                class="poc-pagination"
                @size-change="loadCustomPocsForSelect"
                @current-change="loadCustomPocsForSelect"
              />
            </el-tab-pane>
          </el-tabs>
        </div>

        <!-- 右侧：已选择列表 -->
        <div class="poc-select-right">
          <div class="selected-header">
            <span>{{ $t('task.selected') }} ({{ selectedNucleiTemplates.length + selectedCustomPocs.length }})</span>
            <el-button type="danger" link size="small" @click="clearAllSelections" v-if="selectedNucleiTemplates.length + selectedCustomPocs.length > 0">
              {{ $t('task.clearAll') }}
            </el-button>
          </div>
          <div class="selected-search">
            <el-input v-model="selectedPocSearchKeyword" :placeholder="$t('task.searchSelected')" clearable size="small" :prefix-icon="Search" />
          </div>
          <div class="selected-list">
            <!-- 默认模板 -->
            <div v-if="filteredSelectedNucleiTemplates.length > 0" class="selected-group">
              <div class="group-header">
                <span>{{ $t('task.defaultTemplate') }} ({{ filteredSelectedNucleiTemplates.length }})</span>
                <el-button type="danger" link size="small" @click="clearNucleiSelections">{{ $t('task.clear') }}</el-button>
              </div>
              <div class="selected-items">
                <div v-for="item in filteredSelectedNucleiTemplates" :key="item.id" class="selected-item">
                  <span class="item-name" :title="item.name || item.id">{{ item.name || item.id }}</span>
                  <el-icon class="item-remove" @click="removeNucleiTemplate(item.id)"><Close /></el-icon>
                </div>
              </div>
            </div>
            <!-- 自定义POC -->
            <div v-if="filteredSelectedCustomPocs.length > 0" class="selected-group">
              <div class="group-header">
                <span>{{ $t('task.customPoc') }} ({{ filteredSelectedCustomPocs.length }})</span>
                <el-button type="danger" link size="small" @click="clearCustomPocSelections">{{ $t('task.clear') }}</el-button>
              </div>
              <div class="selected-items">
                <div v-for="item in filteredSelectedCustomPocs" :key="item.id" class="selected-item">
                  <span class="item-name" :title="item.name">{{ item.name }}</span>
                  <el-icon class="item-remove" @click="removeCustomPoc(item.id)"><Close /></el-icon>
                </div>
              </div>
            </div>
            <!-- 空状态 -->
            <div v-if="filteredSelectedNucleiTemplates.length === 0 && filteredSelectedCustomPocs.length === 0" class="selected-empty">
              <span>{{ selectedPocSearchKeyword ? $t('task.noMatchingResults') : $t('task.noPocSelected') }}</span>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <el-button @click="pocSelectDialogVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="confirmPocSelection">{{ $t('common.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- 查看POC内容对话框 -->
    <el-dialog v-model="pocContentDialogVisible" :title="pocContentTitle" width="800px">
      <el-descriptions :column="2" border size="small" style="margin-bottom: 15px">
        <el-descriptions-item :label="$t('task.templateId')">{{ currentViewPoc.id || currentViewPoc.templateId }}</el-descriptions-item>
        <el-descriptions-item :label="$t('common.name')">{{ currentViewPoc.name }}</el-descriptions-item>
        <el-descriptions-item :label="$t('task.severityLevel')">
          <el-tag :type="getSeverityType(currentViewPoc.severity)" size="small">{{ currentViewPoc.severity }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item :label="$t('task.author')">{{ currentViewPoc.author || '-' }}</el-descriptions-item>
      </el-descriptions>
      <div class="poc-content-wrapper" v-loading="pocContentLoading">
        <el-input
          v-model="currentViewPoc.content"
          type="textarea"
          :rows="18"
          readonly
        />
      </div>
      <template #footer>
        <el-button @click="pocContentDialogVisible = false">{{ $t('common.close') }}</el-button>
        <el-button type="primary" @click="copyPocContent">{{ $t('task.copyContent') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, nextTick, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh, Edit, Delete, VideoPlay, Close, Search, InfoFilled } from '@element-plus/icons-vue'
import { 
  getCronTaskList, 
  saveCronTask, 
  toggleCronTask, 
  deleteCronTask,
  batchDeleteCronTask,
  runCronTaskNow,
  validateCronSpec 
} from '@/api/crontask'
import { getTaskList } from '@/api/task'
import { getNucleiTemplateList, getCustomPocList, getNucleiTemplateDetail } from '@/api/poc'
import { getDirScanDictEnabledList } from '@/api/dirscan'
import { getSubdomainDictEnabledList } from '@/api/subdomain'

const router = useRouter()
const { t } = useI18n()
const loading = ref(false)
const tableData = ref([])
const selectedRows = ref([])
const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

const dialogVisible = ref(false)
const isEdit = ref(false)
const submitting = ref(false)
const formRef = ref(null)
const form = reactive({

  id: '',
  name: '',
  scheduleType: 'cron',
  cronSpec: '0 0 2 * * *',
  scheduleTime: '',
  scheduleTimeDate: null,
  mainTaskId: '',
  target: '',
  config: '',
// 子域名扫描
  domainscanEnable: false,
  domainscanSubfinder: true,
  domainscanBruteforce: false, // 字典爆破
  domainscanBruteforceTimeout: 30, // KSubdomain 超时时间（分钟）
  domainscanTimeout: 300,
  domainscanMaxEnumTime: 10,
  domainscanThreads: 10,
  domainscanRateLimit: 0,
  domainscanRemoveWildcard: true,
  domainscanResolveDNS: true,
  domainscanConcurrent: 50,
  subdomainDictIds: [], // 子域名暴力破解字典
  subdomainDicts: [], // 保存已选择的字典信息
  // KSubdomain增强功能
  domainscanRecursiveBrute: false, // 递归爆破
  recursiveDictIds: [], // 递归爆破字典ID列表
  recursiveDicts: [], // 保存已选择的递归字典信息
  domainscanWildcardDetect: true,  // 泛解析检测
      // 端口扫描
  portscanEnable: true,
  portscanTool: 'naabu',
  portscanRate: 3000, // 提高默认值从1000到3000
  ports: 'top100',
  portThreshold: 100,
  scanType: 'c',
  portscanTimeout: 60,
  skipHostDiscovery: false,
  excludeCDN: false,
  excludeHosts: '',
  portscanWorkers: 50, // Naabu内部工作线程，默认50
  portscanRetries: 2, // 重试次数，默认2
  portscanWarmUpTime: 1, // 预热时间(秒)，默认1
  portscanVerify: false, // TCP验证，默认false
  // 端口识别
  portidentifyEnable: false,
  portidentifyTool: 'nmap',
  portidentifyTimeout: 30,
  portidentifyConcurrency: 10,
  portidentifyArgs: '',
  portidentifyUDP: false,
  portidentifyFastMode: false,
  portidentifyForceScan: false,
  // 指纹识别
  fingerprintEnable: true,
  fingerprintTool: 'httpx',
  fingerprintIconHash: true,
  fingerprintCustomEngine: false,
  fingerprintScreenshot: false,
  fingerprintActiveScan: false,
  fingerprintActiveTimeout: 10,
  fingerprintTimeout: 90,
  fingerprintFilterMode: 'http_mapping', // 过滤模式: http_mapping(HTTP映射) 或 service_mapping(服务映射)
  fingerprintForceScan: false,
  // 漏洞扫描
  pocscanEnable: false,
  pocscanMode: 'auto',
  pocscanAutoScan: true,
  pocscanAutomaticScan: true,
  pocscanCustomOnly: false,
  pocscanSeverity: ['critical', 'high', 'medium'],
  pocscanTargetTimeout: 600,
  pocscanRateLimit: 800,
  pocscanConcurrency: 80,
  pocscanForceScan: false,
  pocscanNucleiTemplateIds: [],
  pocscanCustomPocIds: [],
  // 自定义HTTP头部
  pocscanHeaderMode: 'none', // none / preset / custom
  pocscanPresetUA: '',
  pocscanCustomHeadersText: '',
  // 保存已选择的对象信息（用于显示名称）
  pocscanNucleiTemplates: [],
  pocscanCustomPocs: [],
  // 目录扫描
  dirscanEnable: false,
  dirscanDictIds: [],
  dirscanDicts: [], // 保存已选择的字典信息
  dirscanThreads: 50,
  dirscanTimeout: 10,
  dirscanFollowRedirect: false,
  dirscanForceScan: false,
  // ffuf 高级配置
  dirscanAutoCalibration: true,
  dirscanFilterSize: '',
  dirscanFilterWords: '',
  dirscanFilterLines: '',
  dirscanFilterRegex: '',
  dirscanMatcherMode: 'or',
  dirscanFilterMode: 'or',
  dirscanRate: 0,
  dirscanRecursion: false,
  dirscanRecursionDepth: 2
})

// 扫描配置折叠面板
const activeCollapse = ref(['portscan', 'fingerprint'])

// 判断是否有前序扫描阶段启用（用于控制强制扫描开关的显隐）
const hasPrePhaseEnabled = computed(() => {
  return form.domainscanEnable || form.portscanEnable ||
         form.portidentifyEnable || form.fingerprintEnable
})

// 目录扫描字典选择相关
const dictSelectDialogVisible = ref(false)
const dictList = ref([])
const dictLoading = ref(false)
const dictTableRef = ref()
const selectedDictIds = ref([])

// 子域名字典选择相关
const subdomainDictSelectDialogVisible = ref(false)
const subdomainDictList = ref([])
const subdomainDictLoading = ref(false)
const subdomainDictTableRef = ref()
const selectedSubdomainDictIds = ref([])

// 递归爆破字典选择相关
const recursiveDictSelectDialogVisible = ref(false)
const recursiveDictList = ref([])
const recursiveDictLoading = ref(false)
const recursiveDictTableRef = ref()
const selectedRecursiveDictIds = ref([])

// POC选择相关
const pocSelectDialogVisible = ref(false)
const pocSelectTab = ref('nuclei')
const nucleiTemplateList = ref([])
const customPocList = ref([])
const nucleiTemplateLoading = ref(false)
const customPocLoading = ref(false)
const selectAllNucleiLoading = ref(false)
const selectAllCustomLoading = ref(false)
const isSelectingAll = ref(false)
const isLoadingData = ref(false)
const nucleiTableRef = ref()
const customPocTableRef = ref()
const selectedNucleiTemplateIds = ref([])
const selectedCustomPocIds = ref([])
const selectedNucleiTemplates = ref([])
const selectedCustomPocs = ref([])
const selectedPocSearchKeyword = ref('')
const nucleiTemplateFilter = reactive({ keyword: '', severity: '', category: '', tag: '' })
const customPocFilter = reactive({ name: '', severity: '', tag: '' })
const nucleiTemplatePagination = reactive({ page: 1, pageSize: 50, total: 0 })
const customPocPagination = reactive({ page: 1, pageSize: 50, total: 0 })

// 查看POC内容相关
const pocContentDialogVisible = ref(false)
const pocContentLoading = ref(false)
const pocContentTitle = ref('')
const currentViewPoc = ref({})

// 过滤后的已选择列表
const filteredSelectedNucleiTemplates = computed(() => {
  if (!selectedPocSearchKeyword.value) return selectedNucleiTemplates.value
  const keyword = selectedPocSearchKeyword.value.toLowerCase()
  return selectedNucleiTemplates.value.filter(t => 
    (t.name && t.name.toLowerCase().includes(keyword)) || 
    (t.id && t.id.toLowerCase().includes(keyword))
  )
})

const filteredSelectedCustomPocs = computed(() => {
  if (!selectedPocSearchKeyword.value) return selectedCustomPocs.value
  const keyword = selectedPocSearchKeyword.value.toLowerCase()
  return selectedCustomPocs.value.filter(p => 
    (p.name && p.name.toLowerCase().includes(keyword)) || 
    (p.templateId && p.templateId.toLowerCase().includes(keyword)) ||
    (p.id && p.id.toLowerCase().includes(keyword))
  )
})

const rules = {
  name: [{ required: true, message: t('cronTask.pleaseEnterName'), trigger: 'blur' }],
  mainTaskId: [{ required: true, message: t('cronTask.pleaseSelectTask'), trigger: 'change' }],
  scheduleType: [{ required: true, message: t('common.pleaseSelect'), trigger: 'change' }],
  cronSpec: [{ 
    required: true, 
    validator: (rule, value, callback) => {
      if (form.scheduleType === 'cron' && !value) {
        callback(new Error(t('cronTask.cronValidateError')))
      } else {
        callback()
      }
    },
    trigger: 'blur' 
  }],
  scheduleTime: [{
    required: true,
    validator: (rule, value, callback) => {
      if (form.scheduleType === 'once' && !form.scheduleTimeDate) {
        callback(new Error(t('common.pleaseSelect')))
      } else {
        callback()
      }
    },
    trigger: 'change'
  }]
}

const cronPresets = computed(() => [
  { label: t('cronTask.everyHour'), value: '0 0 * * * *' },
  { label: t('cronTask.everyDay2am'), value: '0 0 2 * * *' },
  { label: t('cronTask.everyMonday'), value: '0 0 3 * * 1' },
  { label: t('cronTask.every6hours'), value: '0 0 */6 * * *' }
])

const cronValidation = reactive({
  valid: false,
  nextTimes: [],
  error: ''
})

const taskList = ref([])

const selectedTask = computed(() => {
  return taskList.value.find(t => t.taskId === form.mainTaskId)
})

// 加载数据
async function loadData() {
  loading.value = true
  try {
    const res = await getCronTaskList({
      page: pagination.page,
      pageSize: pagination.pageSize
    })
    // 调试日志已移除
    if (res.code === 0) {
      tableData.value = res.data?.list || []
      pagination.total = res.data?.total || 0
    } else {
      console.error('加载定时任务失败:', res.msg)
    }
  } catch (error) {
    console.error('加载定时任务失败:', error)
  } finally {
    loading.value = false
  }
}

// 加载任务列表（只加载已创建状态的任务作为模板）
async function loadTaskList() {
  try {
    const res = await getTaskList({ page: 1, pageSize: 500 })
    if (res.code === 0) {
      // 过滤出可用的任务（已创建或已完成的任务）
      // 注意：list直接在res下，不是res.data.list
      taskList.value = (res.list || []).filter(t => 
        ['CREATED', 'SUCCESS', 'FAILURE', 'STOPPED'].includes(t.status)
      )
    }
  } catch (error) {
    console.error('加载任务列表失败:', error)
  }
}

// 截取目标显示
function truncateTarget(target, maxLen = 40) {
  if (!target) return ''
  const firstLine = target.split('\n')[0]
  if (firstLine.length > maxLen) {
    return firstLine.substring(0, maxLen) + '...'
  }
  return firstLine
}

// 显示创建对话框
function showCreateDialog() {
  isEdit.value = false
  Object.assign(form, {

    batchSize: config.batchSize || 50,
    // 子域名扫描
    domainscanEnable: config.domainscan?.enable ?? false,
    domainscanSubfinder: config.domainscan?.subfinder ?? true,
    domainscanBruteforce: hasBruteforce,
    domainscanBruteforceTimeout: config.domainscan?.bruteforceTimeout || 30,
    domainscanTimeout: config.domainscan?.timeout || 300,
    domainscanMaxEnumTime: config.domainscan?.maxEnumerationTime || 10,
    domainscanThreads: config.domainscan?.threads || 10,
    domainscanRateLimit: config.domainscan?.rateLimit || 0,
    domainscanRemoveWildcard: config.domainscan?.removeWildcard ?? true,
    domainscanResolveDNS: config.domainscan?.resolveDNS ?? true,
    domainscanConcurrent: config.domainscan?.concurrent || 50,
    subdomainDictIds: config.domainscan?.subdomainDictIds || [],
    // KSubdomain增强功能
    domainscanRecursiveBrute: config.domainscan?.recursiveBrute ?? false,
    recursiveDictIds: config.domainscan?.recursiveDictIds || [],
    domainscanWildcardDetect: config.domainscan?.wildcardDetect ?? true,
    // 端口扫描
    portscanEnable: config.portscan?.enable ?? true,
    portscanTool: config.portscan?.tool || 'naabu',
    portscanRate: config.portscan?.rate || 1000,
    ports: config.portscan?.ports || 'top100',
    portThreshold: config.portscan?.portThreshold || 100,
    scanType: config.portscan?.scanType || 'c',
    portscanTimeout: config.portscan?.timeout || 60,
    skipHostDiscovery: config.portscan?.skipHostDiscovery ?? false,
    excludeCDN: config.portscan?.excludeCDN ?? false,
    excludeHosts: config.portscan?.excludeHosts || '',
    // 端口识别
    portidentifyEnable: config.portidentify?.enable ?? false,
    portidentifyTool: config.portidentify?.tool || 'nmap',
    portidentifyTimeout: config.portidentify?.timeout || 30,
    portidentifyConcurrency: config.portidentify?.concurrency || 10,
    portidentifyArgs: config.portidentify?.args || '',
    portidentifyUDP: config.portidentify?.udp ?? false,
    portidentifyFastMode: config.portidentify?.fastMode ?? false,
    // 指纹识别
    fingerprintEnable: config.fingerprint?.enable ?? true,
    fingerprintTool: config.fingerprint?.tool || (config.fingerprint?.httpx ? 'httpx' : 'builtin'),
    fingerprintIconHash: config.fingerprint?.iconHash ?? true,
    fingerprintCustomEngine: config.fingerprint?.customEngine ?? false,
    fingerprintScreenshot: config.fingerprint?.screenshot ?? false,
    fingerprintActiveScan: config.fingerprint?.activeScan ?? false,
    fingerprintActiveTimeout: config.fingerprint?.activeTimeout || 10,
    fingerprintTimeout: config.fingerprint?.targetTimeout || 90,
    fingerprintFilterMode: config.fingerprint?.filterMode || 'http_mapping',
    // 漏洞扫描
    pocscanEnable: config.pocscan?.enable ?? false,
    pocscanMode: isManualMode ? 'manual' : 'auto',
    pocscanAutoScan: config.pocscan?.autoScan ?? true,
    pocscanAutomaticScan: config.pocscan?.automaticScan ?? true,
    pocscanCustomOnly: config.pocscan?.customPocOnly ?? false,
    pocscanSeverity: config.pocscan?.severity ? config.pocscan.severity.split(',') : ['critical', 'high', 'medium'],
    pocscanTargetTimeout: config.pocscan?.targetTimeout || 600,
    pocscanRateLimit: config.pocscan?.rateLimit || 800,
    pocscanConcurrency: config.pocscan?.concurrency || 80,
    pocscanNucleiTemplateIds: config.pocscan?.nucleiTemplateIds || [],
    pocscanCustomPocIds: config.pocscan?.customPocIds || [],
    ...parseCustomHeaders(config.pocscan?.customHeaders),
    // 目录扫描
    dirscanEnable: config.dirscan?.enable ?? false,
    dirscanDictIds: config.dirscan?.dictIds || [],
    dirscanThreads: config.dirscan?.threads || 50,
    dirscanTimeout: config.dirscan?.timeout || 10,
    dirscanFollowRedirect: config.dirscan?.followRedirect ?? false
  
  })
}

// 重置扫描配置为默认值
function resetScanConfig() {
  // 子域名扫描
  form.domainscanEnable = false
  form.domainscanSubfinder = true
  form.domainscanBruteforce = false, // 字典爆破
  form.domainscanBruteforceTimeout = 30, // KSubdomain 超时时间（分钟）
  form.domainscanTimeout = 300
  form.domainscanMaxEnumTime = 10
  form.domainscanThreads = 10
  form.domainscanRateLimit = 0
  form.domainscanRemoveWildcard = true
  form.domainscanResolveDNS = true
  form.domainscanConcurrent = 50
  form.subdomainDictIds = [], // 子域名暴力破解字典
  form.subdomainDicts = [], // 保存已选择的字典信息
  // KSubdomain增强功能
  form.domainscanRecursiveBrute = false, // 递归爆破
  form.recursiveDictIds = [], // 递归爆破字典ID列表
  form.recursiveDicts = [], // 保存已选择的递归字典信息
  form.domainscanWildcardDetect = true,  // 泛解析检测
      // 端口扫描
  form.portscanEnable = true
  form.portscanTool = 'naabu'
  form.portscanRate = 3000, // 提高默认值从1000到3000
  form.ports = 'top100'
  form.portThreshold = 100
  form.scanType = 'c'
  form.portscanTimeout = 60
  form.skipHostDiscovery = false
  form.excludeCDN = false
  form.excludeHosts = ''
  form.portscanWorkers = 50, // Naabu内部工作线程，默认50
  form.portscanRetries = 2, // 重试次数，默认2
  form.portscanWarmUpTime = 1, // 预热时间(秒)，默认1
  form.portscanVerify = false, // TCP验证，默认false
  // 端口识别
  form.portidentifyEnable = false
  form.portidentifyTool = 'nmap'
  form.portidentifyTimeout = 30
  form.portidentifyConcurrency = 10
  form.portidentifyArgs = ''
  form.portidentifyUDP = false
  form.portidentifyFastMode = false
  form.portidentifyForceScan = false
  // 指纹识别
  form.fingerprintEnable = true
  form.fingerprintTool = 'httpx'
  form.fingerprintIconHash = true
  form.fingerprintCustomEngine = false
  form.fingerprintScreenshot = false
  form.fingerprintActiveScan = false
  form.fingerprintActiveTimeout = 10
  form.fingerprintTimeout = 90
  form.fingerprintFilterMode = 'http_mapping', // 过滤模式: http_mapping(HTTP映射) 或 service_mapping(服务映射)
  form.fingerprintForceScan = false
  // 漏洞扫描
  form.pocscanEnable = false
  form.pocscanMode = 'auto'
  form.pocscanAutoScan = true
  form.pocscanAutomaticScan = true
  form.pocscanCustomOnly = false
  form.pocscanSeverity = ['critical', 'high', 'medium']
  form.pocscanTargetTimeout = 600
  form.pocscanRateLimit = 800
  form.pocscanConcurrency = 80
  form.pocscanForceScan = false
  form.pocscanNucleiTemplateIds = []
  form.pocscanCustomPocIds = []
  // 自定义HTTP头部
  form.pocscanHeaderMode = 'none', // none / preset / custom
  form.pocscanPresetUA = ''
  form.pocscanCustomHeadersText = ''
  // 保存已选择的对象信息（用于显示名称）
  form.pocscanNucleiTemplates = []
  form.pocscanCustomPocs = []
  // 目录扫描
  form.dirscanEnable = false
  form.dirscanDictIds = []
  form.dirscanDicts = [], // 保存已选择的字典信息
  form.dirscanThreads = 50
  form.dirscanTimeout = 10
  form.dirscanFollowRedirect = false
  form.dirscanForceScan = false
  // ffuf 高级配置
  form.dirscanAutoCalibration = true
  form.dirscanFilterSize = ''
  form.dirscanFilterWords = ''
  form.dirscanFilterLines = ''
  form.dirscanFilterRegex = ''
  form.dirscanMatcherMode = 'or'
  form.dirscanFilterMode = 'or'
  form.dirscanRate = 0
  form.dirscanRecursion = false
  dirscanRecursionDepth: 2
  selectedNucleiTemplates.value = []
  selectedCustomPocs.value = []

}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.cron-task-page {
  padding: 20px;
}

.action-card {
  margin-bottom: 20px;
}

.table-card {
  margin-bottom: 20px;
}

.pagination {
  margin-top: 20px;
  justify-content: flex-end;
}

.cron-code {
  background: var(--el-fill-color-light);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 12px;
  margin-left: 6px;
}

.schedule-time {
  font-size: 12px;
  color: var(--el-text-color-regular);
  margin-left: 6px;
}

.text-muted {
  color: var(--el-text-color-placeholder);
}

.task-link {
  color: var(--el-color-primary);
  cursor: pointer;
}

.task-link:hover {
  text-decoration: underline;
}

.task-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.task-option .task-name {
  flex: 1;
}

.task-option .task-target {
  color: var(--el-text-color-placeholder);
  font-size: 12px;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.selected-task-info {
  margin-top: 8px;
}

.cron-help {
  margin-top: 10px;
}

.cron-presets {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 10px;
}

.preset-label {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.preset-tag {
  cursor: pointer;
}

.preset-tag:hover {
  background: var(--el-color-primary-light-7);
}

.cron-next-times {
  background: var(--el-fill-color-lighter);
  padding: 10px;
  border-radius: 4px;
  font-size: 12px;
}

.next-label {
  color: var(--el-text-color-secondary);
  margin-bottom: 5px;
}

.next-time {
  color: var(--el-text-color-regular);
  line-height: 1.8;
}

.cron-error {
  color: var(--el-color-danger);
  font-size: 12px;
}

.form-hint {
  color: var(--el-text-color-placeholder);
  font-size: 12px;
  margin-top: 5px;
}

/* 扫描配置折叠面板样式 */
.config-collapse {
  margin-top: 20px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 4px;
}

.collapse-title {
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 8px;
}

.scan-tools-layout {
  margin-top: 10px;
}

.scan-tool-section {
  background: var(--el-fill-color-lighter);
  border-radius: 6px;
  padding: 15px;
  min-height: 200px;
}

.scan-tool-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.scan-tool-title {
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.scan-tool-disabled-hint {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--el-text-color-placeholder);
  padding: 20px;
  justify-content: center;
}

.selected-dict-summary {
  display: flex;
  align-items: center;
  gap: 10px;
}

.selected-poc-summary {
  display: flex;
  align-items: center;
  gap: 10px;
}

.warning-hint {
  color: var(--el-color-warning);
  font-size: 12px;
}

.secondary-hint {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

/* POC选择对话框样式 */
.poc-select-container {
  display: flex;
  gap: 20px;
  height: 500px;
}

.poc-select-left {
  flex: 1;
  overflow: auto;
}

.poc-select-right {
  width: 280px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 4px;
  display: flex;
  flex-direction: column;
}

.selected-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 15px;
  border-bottom: 1px solid var(--el-border-color-light);
  font-weight: 500;
}

.selected-search {
  padding: 10px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.selected-list {
  flex: 1;
  overflow-y: auto;
  padding: 10px;
}

.selected-group {
  margin-bottom: 15px;
}

.group-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 5px 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.selected-items {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
}

.selected-item {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 3px 8px;
  background: var(--el-fill-color-light);
  border-radius: 3px;
  font-size: 12px;
  max-width: 100%;
}

.item-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 200px;
}

.item-remove {
  cursor: pointer;
  color: var(--el-text-color-placeholder);
  flex-shrink: 0;
}

.item-remove:hover {
  color: var(--el-color-danger);
}

.selected-empty {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100px;
  color: var(--el-text-color-placeholder);
}

.poc-filter-form {
  margin-bottom: 10px;
}

.poc-pagination {
  margin-top: 10px;
  justify-content: flex-end;
}

.poc-content-wrapper {
  min-height: 300px;
}
</style>
