import { act, fireEvent, render } from '@testing-library/react-native';

import { createTask, listTasks } from '@/api/tasks';
import { TaskListScreen } from '@/components/task-list-screen';

jest.mock('@/api/tasks', () => ({
  createTask: jest.fn(),
  listTasks: jest.fn(),
  updateTask: jest.fn(),
}));

const mockedCreateTask = jest.mocked(createTask);
const mockedListTasks = jest.mocked(listTasks);

describe('<TaskListScreen />', () => {
  beforeEach(() => {
    jest.useFakeTimers();
    mockedListTasks.mockResolvedValue({
      data: [],
      meta: { page: 1, limit: 10, total: 0, total_pages: 0 },
    });
    mockedCreateTask.mockResolvedValue({
      id: 1,
      title: 'New task',
      description: '',
      status: 'todo',
      assignee: '',
      created_at: '2026-09-24T00:00:00Z',
      updated_at: '2026-09-24T00:00:00Z',
    });
  });

  afterEach(() => {
    jest.useRealTimers();
    jest.clearAllMocks();
  });

  it('requests tasks with the debounced keyword', async () => {
    const screen = await render(<TaskListScreen />);
    await act(() => jest.advanceTimersByTime(0));
    await act(async () => undefined);
    expect(mockedListTasks).toHaveBeenCalledTimes(1);

    await fireEvent.changeText(screen.getByLabelText('Search tasks'), 'login');
    await act(() => jest.advanceTimersByTime(350));
    await act(() => jest.advanceTimersByTime(0));
    await act(async () => undefined);

    expect(mockedListTasks).toHaveBeenLastCalledWith(
      expect.objectContaining({ keyword: 'login', page: 1 }),
      expect.any(AbortSignal),
    );
  });

  it('creates a task and refreshes the list', async () => {
    const screen = await render(<TaskListScreen />);
    await act(() => jest.advanceTimersByTime(0));
    await act(async () => undefined);

    await fireEvent.press(screen.getByLabelText('Add task'));
    await fireEvent.changeText(screen.getByLabelText('Task title'), 'New task');
    await fireEvent.press(screen.getByLabelText('Save task'));
    await act(async () => undefined);

    expect(mockedCreateTask).toHaveBeenCalledWith({
      title: 'New task',
      description: '',
      status: 'todo',
      assignee: '',
    });

    await act(() => jest.advanceTimersByTime(0));
    await act(async () => undefined);
    expect(mockedListTasks).toHaveBeenCalledTimes(2);
  });
});
