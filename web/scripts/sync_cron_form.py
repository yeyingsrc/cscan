import re
import sys

# 1. Read files
with open('D:/cscan/cscan/web/src/views/TaskCreate.vue', 'r', encoding='utf-8') as f:
    task_create_content = f.read()

with open('D:/cscan/cscan/web/src/views/CronTask.vue', 'r', encoding='utf-8') as f:
    cron_task_content = f.read()

# 2. Extract collapse items from TaskCreate.vue
match_template = re.search(r'(<!-- 子域名扫描 -->\s+<el-collapse-item name="domainscan">.*?</el-collapse-item>\n\s*</el-collapse>)', task_create_content, re.DOTALL)
if not match_template:
    print("Cannot find template in TaskCreate.vue")
    sys.exit(1)

template_str = match_template.group(1)
template_str = template_str.replace('\n\s*</el-collapse>', '') # remove closing tag
template_str = re.sub(r'</el-collapse>\s*$', '', template_str).strip()

# 3. Replace in CronTask.vue
cron_task_content = re.sub(
    r'<!-- 子域名扫描 -->\s+<el-collapse-item name="domainscan">.*?</el-collapse-item>\s*</el-collapse>',
    template_str + '\n        </el-collapse>',
    cron_task_content,
    flags=re.DOTALL
)

# 4. Update reactive form
# TaskCreate has a lot of form defaults. Let's extract the form reactive object
match_form = re.search(r'const form = reactive\(\{(.*?)\}\)', task_create_content, re.DOTALL)
form_body = match_form.group(1)

# we need to keep id, name, scheduleType, cronSpec, scheduleTime, scheduleTimeDate, mainTaskId, target, config
# and then append everything else from form_body
cron_top_vars = """
  id: '',
  name: '',
  scheduleType: 'cron',
  cronSpec: '0 0 2 * * *',
  scheduleTime: '',
  scheduleTimeDate: null,
  mainTaskId: '',
  target: '',
  config: '',
"""

# Extract from domainscanEnable to the end from TaskCreate
match_task_form = re.search(r'(// 子域名扫描\s+domainscanEnable: false,.*?)(?=\n\s*\}\))', task_create_content, re.DOTALL)
rest_of_form = match_task_form.group(1)

new_form = "const form = reactive({\n" + cron_top_vars + rest_of_form + "\n})"
cron_task_content = re.sub(r'const form = reactive\(\{.*?// 高级设置.*?batchSize: 50\n\}\)', new_form, cron_task_content, flags=re.DOTALL)

# 5. Update buildConfig
# Extract buildConfig from TaskCreate
match_build_config = re.search(r'function buildConfig\(\) \{(.*?return config\n\})', task_create_content, re.DOTALL)
build_config_body = match_build_config.group(1)
new_build_config = "function buildConfig() {" + build_config_body

cron_task_content = re.sub(r'function buildConfig\(\) \{.*?return config\n\}', new_build_config, cron_task_content, flags=re.DOTALL)

# 6. Update applyConfig
# Extract applyConfig from TaskCreate
match_apply_config = re.search(r'function applyConfig\(config\) \{.*?Object\.assign\(form, \{(.*?)\}\)\n\}', task_create_content, re.DOTALL)
apply_config_body = match_apply_config.group(1)

# In CronTask.vue, replace the Object.assign in applyConfig
cron_task_content = re.sub(
    r'Object\.assign\(form, \{.*?dirscanFollowRedirect: config\.dirscan\?\.followRedirect \?\? false\n\s*\}\)',
    "Object.assign(form, {\n" + apply_config_body + "\n  })",
    cron_task_content,
    flags=re.DOTALL
)

# 7. Update resetScanConfig
# Find everything from domainscanEnable in resetScanConfig in TaskCreate (we can just use the form ones but with '=')
reset_body = rest_of_form
# convert : to =
reset_body = re.sub(r'(\w+):\s*(.*?),?\n', r'form.\1 = \2\n', reset_body)
# Also need to handle selectedNucleiTemplates etc
extra_resets = "\n  selectedNucleiTemplates.value = []\n  selectedCustomPocs.value = []\n"
cron_task_content = re.sub(
    r'// 子域名扫描\s+form\.domainscanEnable = false;?.*?form\.batchSize = 50',
    reset_body.strip() + extra_resets,
    cron_task_content,
    flags=re.DOTALL
)

# 8. Fix some issues with resetScanConfig syntax
# Remove comments that might have been mangled and fix trailing commas
with open('D:/cscan/cscan/web/src/views/CronTask.vue', 'w', encoding='utf-8') as f:
    f.write(cron_task_content)

print("Patched CronTask.vue")
