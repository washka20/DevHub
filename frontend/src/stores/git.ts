import { useGitStatusStore } from './gitStatus'
import { useGitLogStore } from './gitLog'
import { useGitBranchesStore } from './gitBranches'
import { useGitDiffStore } from './gitDiff'
import { useGitStashStore } from './gitStash'
import { useGitCommitStore } from './gitCommit'
import { computed, ref } from 'vue'

export { useGitStatusStore } from './gitStatus'
export { useGitLogStore } from './gitLog'
export { useGitBranchesStore } from './gitBranches'
export { useGitDiffStore } from './gitDiff'
export { useGitStashStore } from './gitStash'
export { useGitCommitStore } from './gitCommit'

const _activeTab = ref<'changes' | 'log' | 'branches'>('changes')

/**
 * Facade that composes all git sub-stores into a single API.
 * Keeps backward compatibility with existing consumers.
 */
export function useGitStore() {
  const statusStore = useGitStatusStore()
  const logStore = useGitLogStore()
  const branchesStore = useGitBranchesStore()
  const diffStore = useGitDiffStore()
  const stashStore = useGitStashStore()
  const commitStore = useGitCommitStore()

  const loading = computed(() => ({
    status: statusStore.loading.status,
    branches: branchesStore.loadingBranches,
    log: logStore.loadingLog,
    diff: statusStore.loading.diff,
    commit: commitStore.loadingCommit,
    checkout: branchesStore.loadingCheckout,
    pull: commitStore.loadingPull,
    push: commitStore.loadingPush,
    commitDetail: diffStore.loadingCommitDetail,
    commitDiff: diffStore.loadingCommitDiff,
  }))

  return {
    // gitStatus — direct refs for reactivity
    get status() { return statusStore.status },
    set status(v) { statusStore.status = v },
    get selectedFile() { return statusStore.selectedFile },
    set selectedFile(v) { statusStore.selectedFile = v },
    get diff() { return statusStore.diff },
    set diff(v) { statusStore.diff = v },
    get selectedFiles() { return statusStore.selectedFiles },
    set selectedFiles(v) { statusStore.selectedFiles = v },
    get stagedFiles() { return statusStore.stagedFiles },
    get totalModified() { return statusStore.totalModified },
    get totalStaged() { return statusStore.totalStaged },
    toggleSelectFile: statusStore.toggleSelectFile,
    selectAllUnstaged: statusStore.selectAllUnstaged,
    clearSelection: statusStore.clearSelection,
    isSelected: statusStore.isSelected,
    stageSelected: statusStore.stageSelected,
    unstageAll: statusStore.unstageAll,
    isLocallyStaged: statusStore.isLocallyStaged,
    fetchStatus: statusStore.fetchStatus,
    fetchDiff: statusStore.fetchDiff,

    // gitLog
    get log() { return logStore.log },
    set log(v) { logStore.log = v },
    get viewingBranch() { return logStore.viewingBranch },
    set viewingBranch(v) { logStore.viewingBranch = v },
    get graphNodes() { return logStore.graphNodes },
    get metadataMap() { return logStore.metadataMap },
    get metadataLoaded() { return logStore.metadataLoaded },
    set metadataLoaded(v) { logStore.metadataLoaded = v },
    get metadataLoading() { return logStore.metadataLoading },
    get totalCommits() { return logStore.totalCommits },
    fetchLog: logStore.fetchLog,
    fetchGraph: logStore.fetchGraph,
    fetchMetadata: logStore.fetchMetadata,
    getMetadata: logStore.getMetadata,
    setViewingBranch: logStore.setViewingBranch,

    // gitBranches
    get branches() { return branchesStore.branches },
    set branches(v) { branchesStore.branches = v },
    get branchCommits() { return branchesStore.branchCommits },
    set branchCommits(v) { branchesStore.branchCommits = v },
    fetchBranches: branchesStore.fetchBranches,
    checkout: branchesStore.checkout,
    fetchBranchCommits: branchesStore.fetchBranchCommits,

    // gitDiff
    get selectedCommit() { return diffStore.selectedCommit },
    set selectedCommit(v) { diffStore.selectedCommit = v },
    fetchCommitDetail: diffStore.fetchCommitDetail,
    fetchCommitDiff: diffStore.fetchCommitDiff,

    // gitCommit
    get commitMessage() { return commitStore.commitMessage },
    set commitMessage(v) { commitStore.commitMessage = v },
    get generatingMessage() { return commitStore.generatingMessage },
    generateCommitMessage: commitStore.generateCommitMessage,
    commit: commitStore.commit,
    pull: commitStore.pull,
    push: commitStore.push,
    cherryPick: commitStore.cherryPick,

    // gitStash
    get stashEntries() { return stashStore.stashEntries },
    get stashLoading() { return stashStore.stashLoading },
    fetchStash: stashStore.fetchStash,
    stashPush: stashStore.stashPush,
    stashApply: stashStore.stashApply,
    stashPop: stashStore.stashPop,
    stashDrop: stashStore.stashDrop,
    stashDiff: stashStore.stashDiff,

    // Combined loading
    loading,

    // UI state (not domain-specific)
    get activeTab() { return _activeTab.value },
    set activeTab(v) { _activeTab.value = v },
  }
}
