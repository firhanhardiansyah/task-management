import { useCallback, useEffect, useState } from 'react';
import { SymbolView } from 'expo-symbols';
import {
  ActivityIndicator,
  Alert,
  FlatList,
  KeyboardAvoidingView,
  Modal,
  Platform,
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import {
  createTask,
  deleteTask,
  listTasks,
  Task,
  TaskInput,
  TaskStatus,
  updateTask,
} from '@/api/tasks';

const PAGE_SIZE = 10;
const statuses: { label: string; value: '' | TaskStatus }[] = [
  { label: 'All', value: '' },
  { label: 'To do', value: 'todo' },
  { label: 'In progress', value: 'in_progress' },
  { label: 'Done', value: 'done' },
];

const statusPalettes: Record<
  TaskStatus,
  { backgroundColor: string; borderColor: string; color: string }
> = {
  todo: { backgroundColor: '#fff4d6', borderColor: '#f2c66d', color: '#7a4d00' },
  in_progress: { backgroundColor: '#e8f0ff', borderColor: '#9db7ff', color: '#244fbf' },
  done: { backgroundColor: '#e3f7ea', borderColor: '#8dd4a8', color: '#176b3a' },
};

const allStatusPalette = {
  backgroundColor: '#eaddff',
  borderColor: '#65558f',
  color: '#21005d',
};

const material = {
  background: '#fffbfe',
  surface: '#fffbfe',
  surfaceContainer: '#f3edf7',
  surfaceContainerHigh: '#ece6f0',
  outline: '#79747e',
  outlineVariant: '#cac4d0',
  onSurface: '#1d1b20',
  onSurfaceVariant: '#49454f',
  primary: '#65558f',
  onPrimary: '#ffffff',
  primaryContainer: '#eaddff',
  onPrimaryContainer: '#21005d',
  error: '#b3261e',
  errorContainer: '#f9dedc',
};

function getStatusPalette(status: '' | TaskStatus) {
  return status ? statusPalettes[status] : allStatusPalette;
}

export function TaskListScreen() {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [keyword, setKeyword] = useState('');
  const [debouncedKeyword, setDebouncedKeyword] = useState('');
  const [status, setStatus] = useState<'' | TaskStatus>('');
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [refreshVersion, setRefreshVersion] = useState(0);
  const [creating, setCreating] = useState(false);
  const [editing, setEditing] = useState<Task | null>(null);

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedKeyword(keyword.trim()), 350);
    return () => clearTimeout(timer);
  }, [keyword]);

  const loadTasks = useCallback(
    async (signal?: AbortSignal) => {
      setLoading(true);
      setError('');
      try {
        const response = await listTasks(
          { keyword: debouncedKeyword, status, page, limit: PAGE_SIZE },
          signal,
        );
        setTasks(response.data);
        setTotalPages(response.meta.total_pages);
      } catch (caught) {
        if (!(caught instanceof Error && caught.name === 'AbortError')) {
          setError(caught instanceof Error ? caught.message : 'Unable to load tasks');
        }
      } finally {
        if (!signal?.aborted) setLoading(false);
      }
    },
    [debouncedKeyword, page, status],
  );

  useEffect(() => {
    const controller = new AbortController();
    const timer = setTimeout(() => void loadTasks(controller.signal), 0);
    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [loadTasks, refreshVersion]);

  const changeKeyword = (value: string) => {
    setKeyword(value);
    setPage(1);
  };

  const changeStatus = (value: '' | TaskStatus) => {
    setStatus(value);
    setPage(1);
  };

  return (
    <SafeAreaView style={styles.safeArea} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Text style={styles.title}>Tasks</Text>
        <Text style={styles.subtitle}>Plan your work and keep everything moving.</Text>
      </View>

      <View style={styles.filters}>
        <View style={styles.searchBar}>
          <Text style={styles.searchIcon}>⌕</Text>
          <TextInput
            accessibilityLabel="Search tasks"
            autoCapitalize="none"
            onChangeText={changeKeyword}
            placeholder="Search tasks"
            placeholderTextColor={material.onSurfaceVariant}
            returnKeyType="search"
            style={styles.searchInput}
            value={keyword}
          />
          {keyword ? (
            <Pressable accessibilityLabel="Clear search" onPress={() => changeKeyword('')}>
              <Text style={styles.clearSearch}>×</Text>
            </Pressable>
          ) : null}
        </View>
        <FlatList
          contentContainerStyle={styles.statusList}
          data={statuses}
          horizontal
          keyExtractor={(item) => item.value || 'all'}
          renderItem={({ item }) => {
            const palette = getStatusPalette(item.value);
            const selected = status === item.value;
            return (
              <Pressable
                accessibilityRole="button"
                accessibilityState={{ selected }}
                onPress={() => changeStatus(item.value)}
                style={[
                  styles.statusButton,
                  selected && {
                    backgroundColor: palette.backgroundColor,
                    borderColor: palette.borderColor,
                  },
                ]}>
                <Text style={[styles.statusText, selected && { color: palette.color }]}>
                  {selected ? '✓  ' : ''}{item.label}
                </Text>
              </Pressable>
            );
          }}
          showsHorizontalScrollIndicator={false}
        />
      </View>

      {error ? (
        <View style={styles.message}>
          <Text style={styles.errorText}>{error}</Text>
          <Pressable onPress={() => setRefreshVersion((value) => value + 1)}>
            <Text style={styles.retryText}>Try again</Text>
          </Pressable>
        </View>
      ) : null}

      {loading ? (
        <View style={styles.loading}>
          <ActivityIndicator color={material.primary} size="large" />
          <Text style={styles.loadingText}>Loading tasks…</Text>
        </View>
      ) : (
        <FlatList
          contentContainerStyle={styles.taskList}
          data={tasks}
          keyExtractor={(item) => String(item.id)}
          ListEmptyComponent={
            <View style={styles.emptyState}>
              <View style={styles.emptyIconContainer}><Text style={styles.emptyIcon}>✓</Text></View>
              <Text style={styles.emptyTitle}>Nothing here yet</Text>
              <Text style={styles.emptyText}>Add a task or try another search and filter.</Text>
            </View>
          }
          renderItem={({ item }) => <TaskCard task={item} onEdit={() => setEditing(item)} />}
        />
      )}

      <View style={styles.pagination}>
        <Pressable
          accessibilityLabel="Previous page"
          disabled={page <= 1 || loading}
          onPress={() => setPage((value) => Math.max(1, value - 1))}
          style={[styles.pageButton, (page <= 1 || loading) && styles.buttonDisabled]}>
          <Text style={styles.pageButtonText}>‹  Previous</Text>
        </Pressable>
        <Text style={styles.pageText}>Page {page} of {Math.max(totalPages, 1)}</Text>
        <Pressable
          accessibilityLabel="Next page"
          disabled={page >= totalPages || loading}
          onPress={() => setPage((value) => value + 1)}
          style={[styles.pageButton, (page >= totalPages || loading) && styles.buttonDisabled]}>
          <Text style={styles.pageButtonText}>Next  ›</Text>
        </Pressable>
      </View>

      <Pressable
        accessibilityLabel="Add task"
        accessibilityRole="button"
        onPress={() => setCreating(true)}
        style={({ pressed }) => [styles.fab, pressed && styles.fabPressed]}>
        <Text style={styles.fabText}>＋</Text>
      </Pressable>

      {creating ? (
        <TaskFormModal
          onClose={() => setCreating(false)}
          onSaved={() => {
            setCreating(false);
            setPage(1);
            setRefreshVersion((value) => value + 1);
          }}
        />
      ) : null}

      {editing ? (
        <TaskFormModal
          key={editing.id}
          task={editing}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null);
            setRefreshVersion((value) => value + 1);
          }}
          onDeleted={() => {
            setEditing(null);
            if (tasks.length === 1 && page > 1) {
              setPage((value) => value - 1);
            } else {
              setRefreshVersion((value) => value + 1);
            }
          }}
        />
      ) : null}
    </SafeAreaView>
  );
}

function TaskCard({ task, onEdit }: { task: Task; onEdit: () => void }) {
  const statusLabel = statuses.find((item) => item.value === task.status)?.label ?? task.status;
  const palette = statusPalettes[task.status];
  return (
    <Pressable
      accessibilityLabel={`Edit ${task.title}`}
      onPress={onEdit}
      style={({ pressed }) => [styles.card, pressed && styles.cardPressed]}>
      <View style={styles.cardTop}>
        <Text numberOfLines={2} style={styles.cardTitle}>{task.title}</Text>
        <View
          style={[
            styles.badge,
            { backgroundColor: palette.backgroundColor, borderColor: palette.borderColor },
          ]}>
          <Text style={[styles.badgeText, { color: palette.color }]}>{statusLabel}</Text>
        </View>
      </View>
      {task.description ? <Text numberOfLines={3} style={styles.description}>{task.description}</Text> : null}
      <View style={styles.cardFooter}>
        <Text style={styles.assignee}>{task.assignee ? `Assignee · ${task.assignee}` : 'Unassigned'}</Text>
      </View>
    </Pressable>
  );
}

function TaskFormModal({
  task,
  onClose,
  onSaved,
  onDeleted,
}: {
  task?: Task;
  onClose: () => void;
  onSaved: () => void;
  onDeleted?: () => void;
}) {
  const [form, setForm] = useState<TaskInput>({
    title: task?.title ?? '',
    description: task?.description ?? '',
    status: task?.status ?? 'todo',
    assignee: task?.assignee ?? '',
  });
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState('');
  const isEditing = task !== undefined;
  const busy = saving || deleting;

  const save = async () => {
    if (!form.title.trim()) {
      setError('Title is required');
      return;
    }
    setSaving(true);
    setError('');
    try {
      const input = {
        ...form,
        title: form.title.trim(),
        description: form.description.trim(),
        assignee: form.assignee.trim(),
      };
      if (task) {
        await updateTask(task.id, input);
      } else {
        await createTask(input);
      }
      onSaved();
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : `Unable to ${isEditing ? 'update' : 'create'} task`);
    } finally {
      setSaving(false);
    }
  };

  const remove = async () => {
    if (!task || !onDeleted) return;
    setDeleting(true);
    setError('');
    try {
      await deleteTask(task.id);
      onDeleted();
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to delete task');
    } finally {
      setDeleting(false);
    }
  };

  const confirmDelete = () => {
    if (!task) return;
    Alert.alert(
      'Delete task?',
      `“${task.title}” will be removed from your task list.`,
      [
        { text: 'Cancel', style: 'cancel' },
        { text: 'Delete', style: 'destructive', onPress: () => void remove() },
      ],
    );
  };

  return (
    <Modal animationType="slide" onRequestClose={onClose} transparent visible>
      <KeyboardAvoidingView behavior={Platform.OS === 'ios' ? 'padding' : undefined} style={styles.modalBackdrop}>
        <View style={styles.modalCard}>
          <View style={styles.sheetHandle} />
          <View style={styles.modalHeader}>
            <Text style={styles.modalTitle}>{isEditing ? 'Edit task' : 'Add task'}</Text>
            {isEditing ? (
              <Pressable
                accessibilityLabel="Delete task"
                accessibilityRole="button"
                disabled={busy}
                hitSlop={8}
                onPress={confirmDelete}
                style={({ pressed }) => [
                  styles.deleteIconButton,
                  pressed && styles.deleteIconButtonPressed,
                  busy && styles.buttonDisabled,
                ]}>
                {deleting ? (
                  <ActivityIndicator color={material.error} size="small" />
                ) : (
                  <SymbolView
                    fallback={<Text style={styles.deleteIconFallback}>×</Text>}
                    name={{ ios: 'trash', android: 'delete', web: 'delete' }}
                    size={22}
                    tintColor={material.error}
                  />
                )}
              </Pressable>
            ) : null}
          </View>
          <Text style={styles.fieldLabel}>Title</Text>
          <TextInput accessibilityLabel="Task title" onChangeText={(title) => setForm((value) => ({ ...value, title }))} placeholder="What needs to be done?" placeholderTextColor={material.onSurfaceVariant} style={styles.input} value={form.title} />
          <Text style={styles.fieldLabel}>Description</Text>
          <TextInput accessibilityLabel="Task description" multiline onChangeText={(description) => setForm((value) => ({ ...value, description }))} placeholder="Add details" placeholderTextColor={material.onSurfaceVariant} style={[styles.input, styles.textArea]} value={form.description} />
          <Text style={styles.fieldLabel}>Assignee</Text>
          <TextInput accessibilityLabel="Task assignee" onChangeText={(assignee) => setForm((value) => ({ ...value, assignee }))} placeholder="Name or ID" placeholderTextColor={material.onSurfaceVariant} style={styles.input} value={form.assignee} />
          <Text style={styles.fieldLabel}>Status</Text>
          <View style={styles.modalStatuses}>
            {statuses.slice(1).map((item) => {
              const palette = getStatusPalette(item.value);
              const selected = form.status === item.value;
              return (
                <Pressable
                  key={item.value}
                  onPress={() =>
                    setForm((value) => ({ ...value, status: item.value as TaskStatus }))
                  }
                  style={[
                    styles.statusButton,
                    selected && {
                      backgroundColor: palette.backgroundColor,
                      borderColor: palette.borderColor,
                    },
                  ]}>
                  <Text style={[styles.statusText, selected && { color: palette.color }]}>
                    {item.label}
                  </Text>
                </Pressable>
              );
            })}
          </View>
          {error ? <Text style={styles.errorText}>{error}</Text> : null}
          <View style={styles.modalActions}>
            <Pressable disabled={busy} onPress={onClose} style={styles.cancelButton}><Text style={styles.cancelText}>Cancel</Text></Pressable>
            <Pressable accessibilityLabel="Save task" disabled={busy} onPress={save} style={[styles.saveButton, busy && styles.buttonDisabled]}>
              {saving ? <ActivityIndicator color={material.onPrimary} /> : <Text style={styles.saveText}>{isEditing ? 'Save changes' : 'Create task'}</Text>}
            </Pressable>
          </View>
        </View>
      </KeyboardAvoidingView>
    </Modal>
  );
}

const styles = StyleSheet.create({
  safeArea: { flex: 1, backgroundColor: material.background },
  header: {
    backgroundColor: material.surface,
    paddingBottom: 12,
    paddingHorizontal: 24,
    paddingTop: 20,
  },
  title: { color: material.onSurface, fontSize: 32, fontWeight: '500', letterSpacing: -0.4 },
  subtitle: { color: material.onSurfaceVariant, fontSize: 14, lineHeight: 20, marginTop: 4 },
  filters: {
    backgroundColor: material.surface,
    borderBottomColor: material.outlineVariant,
    borderBottomWidth: StyleSheet.hairlineWidth,
    gap: 14,
    paddingBottom: 16,
    paddingHorizontal: 16,
  },
  searchBar: {
    alignItems: 'center',
    backgroundColor: material.surfaceContainerHigh,
    borderRadius: 28,
    flexDirection: 'row',
    height: 56,
    paddingHorizontal: 18,
  },
  searchIcon: { color: material.onSurface, fontSize: 28, lineHeight: 30, marginRight: 10 },
  searchInput: { color: material.onSurface, flex: 1, fontSize: 16, paddingVertical: 0 },
  clearSearch: { color: material.onSurfaceVariant, fontSize: 28, lineHeight: 30, paddingLeft: 10 },
  statusList: { gap: 8, paddingHorizontal: 1 },
  statusButton: {
    borderColor: material.outline,
    borderRadius: 9,
    borderWidth: 1,
    justifyContent: 'center',
    minHeight: 36,
    paddingHorizontal: 16,
  },
  statusText: { color: material.onSurfaceVariant, fontSize: 14, fontWeight: '600' },
  loading: { flex: 1, alignItems: 'center', justifyContent: 'center', gap: 10 },
  loadingText: { color: material.onSurfaceVariant },
  taskList: { flexGrow: 1, gap: 12, padding: 16, paddingBottom: 88 },
  card: {
    backgroundColor: material.surface,
    borderColor: material.outlineVariant,
    borderRadius: 16,
    borderWidth: StyleSheet.hairlineWidth,
    elevation: 1,
    padding: 16,
    shadowColor: material.onSurface,
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.12,
    shadowRadius: 3,
  },
  cardPressed: { backgroundColor: material.surfaceContainer },
  cardTop: { alignItems: 'flex-start', flexDirection: 'row', gap: 12 },
  cardTitle: { color: material.onSurface, flex: 1, fontSize: 18, fontWeight: '600', lineHeight: 24 },
  badge: { borderRadius: 8, borderWidth: 1, paddingHorizontal: 10, paddingVertical: 5 },
  badgeText: { fontSize: 11, fontWeight: '700', letterSpacing: 0.2 },
  description: { color: material.onSurfaceVariant, lineHeight: 20, marginTop: 10 },
  cardFooter: { alignItems: 'center', flexDirection: 'row', marginTop: 16 },
  assignee: { color: material.onSurfaceVariant, fontSize: 12, fontWeight: '500' },
  emptyState: { alignItems: 'center', paddingHorizontal: 28, paddingTop: 64 },
  emptyIconContainer: { alignItems: 'center', backgroundColor: material.primaryContainer, borderRadius: 36, height: 72, justifyContent: 'center', width: 72 },
  emptyIcon: { color: material.onPrimaryContainer, fontSize: 30 },
  emptyTitle: { color: material.onSurface, fontSize: 20, fontWeight: '600', marginTop: 20 },
  emptyText: { color: material.onSurfaceVariant, lineHeight: 20, marginTop: 6, textAlign: 'center' },
  message: { alignItems: 'center', backgroundColor: material.errorContainer, flexDirection: 'row', justifyContent: 'space-between', margin: 12, padding: 14, borderRadius: 12 },
  errorText: { color: material.error, flexShrink: 1 },
  retryText: { color: material.error, fontWeight: '700', paddingLeft: 16 },
  pagination: { alignItems: 'center', backgroundColor: material.surface, borderTopColor: material.outlineVariant, borderTopWidth: StyleSheet.hairlineWidth, flexDirection: 'row', justifyContent: 'space-between', paddingHorizontal: 16, paddingVertical: 12 },
  pageButton: { backgroundColor: material.primaryContainer, borderRadius: 20, minWidth: 94, paddingHorizontal: 16, paddingVertical: 10 },
  pageButtonText: { color: material.onPrimaryContainer, fontSize: 13, fontWeight: '700', textAlign: 'center' },
  pageText: { color: material.onSurfaceVariant, fontSize: 12, fontWeight: '500' },
  buttonDisabled: { opacity: 0.38 },
  fab: { alignItems: 'center', backgroundColor: material.primaryContainer, borderRadius: 16, bottom: 78, elevation: 6, height: 56, justifyContent: 'center', position: 'absolute', right: 20, shadowColor: material.onSurface, shadowOffset: { width: 0, height: 4 }, shadowOpacity: 0.25, shadowRadius: 5, width: 56 },
  fabPressed: { opacity: 0.86, transform: [{ scale: 0.97 }] },
  fabText: { color: material.onPrimaryContainer, fontSize: 28, fontWeight: '400', lineHeight: 32 },
  modalBackdrop: { backgroundColor: 'rgba(29, 27, 32, 0.42)', flex: 1, justifyContent: 'flex-end' },
  modalCard: { backgroundColor: material.surface, borderTopLeftRadius: 28, borderTopRightRadius: 28, gap: 8, paddingBottom: 28, paddingHorizontal: 24, paddingTop: 10 },
  sheetHandle: { alignSelf: 'center', backgroundColor: material.outlineVariant, borderRadius: 2, height: 4, marginBottom: 12, width: 32 },
  modalHeader: { alignItems: 'center', flexDirection: 'row', justifyContent: 'space-between', marginBottom: 12 },
  modalTitle: { color: material.onSurface, fontSize: 24, fontWeight: '500' },
  deleteIconButton: { alignItems: 'center', backgroundColor: material.errorContainer, borderRadius: 20, height: 40, justifyContent: 'center', width: 40 },
  deleteIconButtonPressed: { opacity: 0.8 },
  deleteIconFallback: { color: material.error, fontSize: 24, lineHeight: 26 },
  fieldLabel: { color: material.onSurfaceVariant, fontSize: 12, fontWeight: '600', marginLeft: 4, marginTop: 4 },
  input: { backgroundColor: material.surface, borderColor: material.outline, borderRadius: 4, borderWidth: 1, color: material.onSurface, fontSize: 16, minHeight: 52, paddingHorizontal: 16, paddingVertical: 12 },
  textArea: { minHeight: 88, textAlignVertical: 'top' },
  modalStatuses: { flexDirection: 'row', flexWrap: 'wrap', gap: 8, marginBottom: 4 },
  modalActions: { flexDirection: 'row', gap: 8, justifyContent: 'flex-end', marginTop: 12 },
  cancelButton: { borderRadius: 20, paddingHorizontal: 18, paddingVertical: 11 },
  cancelText: { color: material.primary, fontWeight: '700' },
  saveButton: { alignItems: 'center', backgroundColor: material.primary, borderRadius: 20, minWidth: 132, paddingHorizontal: 20, paddingVertical: 11 },
  saveText: { color: material.onPrimary, fontWeight: '700' },
});
