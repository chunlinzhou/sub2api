<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-end">
          <div class="flex flex-wrap items-center gap-3">
            <button type="button" class="btn btn-secondary" @click="triggerUpload">
              {{ t('admin.skills.uploadFile') }}
            </button>
            <button type="button" class="btn btn-primary" @click="createNewSkill">
              {{ t('admin.skills.newSkill') }}
            </button>
          </div>
          <input
            ref="uploadInputRef"
            type="file"
            accept=".md,.txt,text/markdown,text/plain"
            class="hidden"
            @change="handleUploadFileChange"
          />
        </div>
      </template>

      <template #table>
        <div class="grid gap-4 xl:grid-cols-[320px_minmax(0,1fr)]">
          <div class="rounded-2xl border border-gray-200 bg-white dark:border-dark-600 dark:bg-dark-800">
            <div class="border-b border-gray-200 px-4 py-3 dark:border-dark-600">
              <div class="flex items-center justify-between gap-3">
                <div class="text-sm font-medium text-gray-900 dark:text-white">
                  {{ t('admin.skills.skillList') }}
                </div>
                <button
                  type="button"
                  class="text-sm text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
                  :disabled="skillsLoading"
                  @click="loadSkills"
                >
                  {{ t('admin.skills.refresh') }}
                </button>
              </div>
            </div>

            <div v-if="skillsLoading" class="px-4 py-6 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.skills.loading') }}
            </div>
            <div v-else-if="!skills.length" class="px-4 py-6 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.skills.empty') }}
            </div>
            <div v-else class="max-h-[70vh] overflow-y-auto p-2">
              <button
                v-for="skill in skills"
                :key="skill.id"
                type="button"
                class="mb-2 w-full rounded-xl border px-3 py-3 text-left transition-colors"
                :class="selectedSkillId === skill.id
                  ? 'border-primary-500 bg-primary-50 dark:border-primary-500 dark:bg-primary-900/20'
                  : 'border-gray-200 hover:bg-gray-50 dark:border-dark-600 dark:hover:bg-dark-700'"
                @click="selectSkill(skill.id)"
              >
                <div class="truncate text-sm font-medium text-gray-900 dark:text-white">
                  {{ skill.filename }}
                </div>
                <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  {{ formatSize(skill.size) }} · {{ formatTime(skill.updated_at) }}
                </div>
              </button>
            </div>
          </div>

          <div class="rounded-2xl border border-gray-200 bg-white dark:border-dark-600 dark:bg-dark-800">
            <div class="border-b border-gray-200 px-4 py-3 dark:border-dark-600">
              <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
                <div class="flex-1">
                  <input
                    v-model="editor.filename"
                    type="text"
                    class="input"
                    :placeholder="t('admin.skills.filenamePlaceholder')"
                  />
                  <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.skills.filenameHint') }}
                  </p>
                </div>
                <div class="flex flex-wrap items-center gap-2">
                  <button
                    type="button"
                    class="btn btn-secondary"
                    :disabled="!selectedSkillId || deleteSubmitting"
                    @click="confirmDeleteCurrent"
                  >
                    {{ deleteSubmitting ? t('admin.skills.deleting') : t('admin.skills.delete') }}
                  </button>
                  <button
                    type="button"
                    class="btn btn-primary"
                    :disabled="saveSubmitting"
                    @click="saveCurrentSkill"
                  >
                    {{ saveSubmitting ? t('common.saving') : t('common.save') }}
                  </button>
                </div>
              </div>
            </div>

            <div class="border-b border-gray-200 px-4 py-3 dark:border-dark-600">
              <div class="flex items-center gap-2">
                <button
                  type="button"
                  class="rounded-lg px-3 py-1.5 text-sm font-medium"
                  :class="activeTab === 'edit'
                    ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300'
                    : 'text-gray-500 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-dark-700'"
                  @click="activeTab = 'edit'"
                >
                  {{ t('admin.skills.editTab') }}
                </button>
                <button
                  type="button"
                  class="rounded-lg px-3 py-1.5 text-sm font-medium"
                  :class="activeTab === 'preview'
                    ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300'
                    : 'text-gray-500 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-dark-700'"
                  @click="activeTab = 'preview'"
                >
                  {{ t('admin.skills.previewTab') }}
                </button>
              </div>
            </div>

            <div v-if="detailLoading" class="px-4 py-8 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.skills.loadingDetail') }}
            </div>
            <div v-else-if="activeTab === 'edit'" class="p-4">
              <textarea
                v-model="editor.content"
                class="input min-h-[60vh] font-mono text-sm"
                :placeholder="t('admin.skills.contentPlaceholder')"
              ></textarea>
            </div>
            <div v-else class="p-4">
              <div
                class="markdown-body prose prose-sm max-w-none break-words dark:prose-invert"
                v-html="previewHtml"
              ></div>
            </div>
          </div>
        </div>
      </template>
    </TablePageLayout>

    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('admin.skills.deleteTitle')"
      :message="t('admin.skills.deleteConfirm', { filename: editor.filename || selectedSkillId || '' })"
      :confirm-text="t('admin.skills.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="deleteCurrentSkill"
      @cancel="showDeleteDialog = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import type { LocalSkillSummary } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'

const { t } = useI18n()
const appStore = useAppStore()

marked.setOptions({
  breaks: true,
  gfm: true
})

const uploadInputRef = ref<HTMLInputElement | null>(null)
const skills = ref<LocalSkillSummary[]>([])
const skillsLoading = ref(false)
const detailLoading = ref(false)
const saveSubmitting = ref(false)
const deleteSubmitting = ref(false)
const showDeleteDialog = ref(false)
const selectedSkillId = ref<string | null>(null)
const activeTab = ref<'edit' | 'preview'>('edit')

const editor = ref({
  filename: '',
  content: ''
})

const previewHtml = computed(() => {
  const html = marked.parse(editor.value.content || '') as string
  return DOMPurify.sanitize(html)
})

async function loadSkills(): Promise<void> {
  skillsLoading.value = true
  try {
    skills.value = await adminAPI.groups.listLocalSkills()
    if (selectedSkillId.value && !skills.value.some((skill) => skill.id === selectedSkillId.value)) {
      selectedSkillId.value = null
    }
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.skills.loadFailed'))
  } finally {
    skillsLoading.value = false
  }
}

async function selectSkill(id: string): Promise<void> {
  selectedSkillId.value = id
  detailLoading.value = true
  activeTab.value = 'edit'
  try {
    const detail = await adminAPI.groups.getLocalSkill(id)
    editor.value = {
      filename: detail.filename,
      content: detail.content
    }
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.skills.loadDetailFailed'))
  } finally {
    detailLoading.value = false
  }
}

function createNewSkill(): void {
  selectedSkillId.value = null
  activeTab.value = 'edit'
  editor.value = {
    filename: '',
    content: ''
  }
}

function triggerUpload(): void {
  uploadInputRef.value?.click()
}

async function handleUploadFileChange(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return

  saveSubmitting.value = true
  try {
    const saved = await adminAPI.groups.uploadLocalSkill(file)
    appStore.showSuccess(t('admin.skills.uploadSuccess'))
    await loadSkills()
    await selectSkill(saved.id)
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.skills.uploadFailed'))
  } finally {
    saveSubmitting.value = false
    input.value = ''
  }
}

async function saveCurrentSkill(): Promise<void> {
  if (!editor.value.filename.trim()) {
    appStore.showError(t('admin.skills.filenameRequired'))
    return
  }

  saveSubmitting.value = true
  try {
    const saved = await adminAPI.groups.saveLocalSkillContent(
      editor.value.filename.trim(),
      editor.value.content
    )
    appStore.showSuccess(t('admin.skills.saveSuccess'))
    await loadSkills()
    await selectSkill(saved.id)
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.skills.saveFailed'))
  } finally {
    saveSubmitting.value = false
  }
}

function confirmDeleteCurrent(): void {
  if (!selectedSkillId.value) {
    return
  }
  showDeleteDialog.value = true
}

async function deleteCurrentSkill(): Promise<void> {
  if (!selectedSkillId.value) {
    return
  }

  deleteSubmitting.value = true
  try {
    await adminAPI.groups.deleteLocalSkill(selectedSkillId.value)
    appStore.showSuccess(t('admin.skills.deleteSuccess'))
    showDeleteDialog.value = false
    createNewSkill()
    await loadSkills()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.skills.deleteFailed'))
  } finally {
    deleteSubmitting.value = false
  }
}

function formatTime(value: string): string {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

function formatSize(size: number): string {
  if (size >= 1024 * 1024) return `${(size / 1024 / 1024).toFixed(1)} MB`
  if (size >= 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${size} B`
}

onMounted(() => {
  loadSkills()
})
</script>

<style scoped>
.markdown-body :deep(pre) {
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
