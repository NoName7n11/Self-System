import { useMemo } from "react";

import { useResourceStore } from "../../stores/useResourceStore";
import { useTaskStore } from "../../stores/useTaskStore";

function formatDateTime(value: string): string {
  if (value.trim() === "") {
    return "No time set";
  }

  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return "Invalid time";
  }

  return parsed.toLocaleString();
}

export default function TaskBoard() {
  const resources = useResourceStore((state) => state.resources);
  const todos = useTaskStore((state) => state.todos);
  const reminders = useTaskStore((state) => state.reminders);
  const isLoadingTodos = useTaskStore((state) => state.isLoadingTodos);
  const isLoadingReminders = useTaskStore((state) => state.isLoadingReminders);
  const selectedTodoId = useTaskStore((state) => state.selectedTodoId);
  const selectedReminderId = useTaskStore((state) => state.selectedReminderId);
  const todoDraft = useTaskStore((state) => state.todoDraft);
  const reminderDraft = useTaskStore((state) => state.reminderDraft);
  const error = useTaskStore((state) => state.error);

  const selectTodo = useTaskStore((state) => state.selectTodo);
  const selectReminder = useTaskStore((state) => state.selectReminder);
  const updateTodoDraft = useTaskStore((state) => state.updateTodoDraft);
  const updateReminderDraft = useTaskStore((state) => state.updateReminderDraft);
  const resetTodoDraft = useTaskStore((state) => state.resetTodoDraft);
  const resetReminderDraft = useTaskStore((state) => state.resetReminderDraft);
  const addTodo = useTaskStore((state) => state.addTodo);
  const updateSelectedTodo = useTaskStore((state) => state.updateSelectedTodo);
  const deleteSelectedTodo = useTaskStore((state) => state.deleteSelectedTodo);
  const setSelectedTodoStatus = useTaskStore((state) => state.setSelectedTodoStatus);
  const addReminder = useTaskStore((state) => state.addReminder);
  const updateSelectedReminder = useTaskStore((state) => state.updateSelectedReminder);
  const deleteSelectedReminder = useTaskStore((state) => state.deleteSelectedReminder);
  const setSelectedReminderStatus = useTaskStore((state) => state.setSelectedReminderStatus);

  const todoCounters = useMemo(() => {
    return {
      open: todos.filter((item) => item.status === "open").length,
      inProgress: todos.filter((item) => item.status === "in_progress").length,
      done: todos.filter((item) => item.status === "done").length,
    };
  }, [todos]);

  const reminderCounters = useMemo(() => {
    return {
      scheduled: reminders.filter((item) => item.status === "scheduled").length,
      sent: reminders.filter((item) => item.status === "sent").length,
      dismissed: reminders.filter((item) => item.status === "dismissed").length,
    };
  }, [reminders]);

  const resourceOptions = useMemo(() => {
    return resources.map((resource) => ({
      id: resource.id,
      label: resource.title.trim() || resource.host.trim() || resource.url.trim() || resource.id,
    }));
  }, [resources]);

  return (
    <section className="task-board panel">
      <div className="panel-heading">
        <h2>Task Operations</h2>
        <p>Create and manage todos and reminders with status controls.</p>
      </div>

      {error ? <p className="error-copy">{error}</p> : null}

      <div className="task-grid">
        <article className="task-card">
          <header className="task-card-heading">
            <h3>Todos</h3>
            <p>
              {todoCounters.open} open • {todoCounters.inProgress} in progress • {todoCounters.done} done
            </p>
          </header>

          <label>
            Title
            <input
              onChange={(event) => updateTodoDraft("title", event.target.value)}
              placeholder="Plan sync rollout"
              value={todoDraft.title}
            />
          </label>

          <label>
            Details
            <textarea
              onChange={(event) => updateTodoDraft("details", event.target.value)}
              placeholder="Include edge-case replay scenarios"
              rows={3}
              value={todoDraft.details}
            />
          </label>

          <div className="task-fields-inline">
            <label>
              Due at
              <input
                onChange={(event) => updateTodoDraft("dueAt", event.target.value)}
                type="datetime-local"
                value={todoDraft.dueAt}
              />
            </label>

            <label>
              Status
              <select onChange={(event) => updateTodoDraft("status", event.target.value)} value={todoDraft.status}>
                <option value="open">Open</option>
                <option value="in_progress">In progress</option>
                <option value="done">Done</option>
              </select>
            </label>
          </div>

          <label>
            Linked resource
            <select onChange={(event) => updateTodoDraft("resourceId", event.target.value)} value={todoDraft.resourceId}>
              <option value="">None</option>
              {resourceOptions.map((option) => (
                <option key={option.id} value={option.id}>
                  {option.label}
                </option>
              ))}
            </select>
          </label>

          <div className="form-actions">
            <button className="primary-button" onClick={() => void addTodo()} type="button">
              Add Todo
            </button>
            <button className="ghost-button" disabled={!selectedTodoId} onClick={() => void updateSelectedTodo()} type="button">
              Update
            </button>
            <button
              className="ghost-button"
              disabled={!selectedTodoId}
              onClick={() => void setSelectedTodoStatus("done")}
              type="button"
            >
              Mark Done
            </button>
            <button className="ghost-button" disabled={!selectedTodoId} onClick={() => void deleteSelectedTodo()} type="button">
              Delete
            </button>
            <button className="ghost-button" onClick={resetTodoDraft} type="button">
              Clear
            </button>
          </div>

          <div className="task-list-scroll">
            {isLoadingTodos && todos.length === 0 ? <p className="muted-copy">Loading todos...</p> : null}
            {!isLoadingTodos && todos.length === 0 ? <p className="muted-copy">No todos yet.</p> : null}

            {todos.map((todo) => (
              <button
                key={todo.id}
                className={`task-row ${selectedTodoId === todo.id ? "is-selected" : ""}`}
                onClick={() => selectTodo(todo.id)}
                type="button"
              >
                <div className="task-row-main">
                  <h4>{todo.title}</h4>
                  <p>{todo.details || "No details"}</p>
                </div>
                <div className="task-row-meta">
                  <span className="resource-chip">{todo.status.replace("_", " ")}</span>
                  <span>{formatDateTime(todo.dueAt)}</span>
                </div>
                {todo.resourceId ? <div className="task-row-link">Resource: {todo.resourceId}</div> : null}
              </button>
            ))}
          </div>
        </article>

        <article className="task-card">
          <header className="task-card-heading">
            <h3>Reminders</h3>
            <p>
              {reminderCounters.scheduled} scheduled • {reminderCounters.sent} sent • {reminderCounters.dismissed} dismissed
            </p>
          </header>

          <label>
            Title
            <input
              onChange={(event) => updateReminderDraft("title", event.target.value)}
              placeholder="Follow up with dependency check"
              value={reminderDraft.title}
            />
          </label>

          <label>
            Message
            <textarea
              onChange={(event) => updateReminderDraft("message", event.target.value)}
              placeholder="Review rollout evidence and update timeline"
              rows={3}
              value={reminderDraft.message}
            />
          </label>

          <div className="task-fields-inline">
            <label>
              Remind at
              <input
                onChange={(event) => updateReminderDraft("remindAt", event.target.value)}
                type="datetime-local"
                value={reminderDraft.remindAt}
              />
            </label>

            <label>
              Status
              <select onChange={(event) => updateReminderDraft("status", event.target.value)} value={reminderDraft.status}>
                <option value="scheduled">Scheduled</option>
                <option value="sent">Sent</option>
                <option value="dismissed">Dismissed</option>
              </select>
            </label>
          </div>

          <label>
            Linked resource
            <select onChange={(event) => updateReminderDraft("resourceId", event.target.value)} value={reminderDraft.resourceId}>
              <option value="">None</option>
              {resourceOptions.map((option) => (
                <option key={option.id} value={option.id}>
                  {option.label}
                </option>
              ))}
            </select>
          </label>

          <div className="form-actions">
            <button className="primary-button" onClick={() => void addReminder()} type="button">
              Add Reminder
            </button>
            <button
              className="ghost-button"
              disabled={!selectedReminderId}
              onClick={() => void updateSelectedReminder()}
              type="button"
            >
              Update
            </button>
            <button
              className="ghost-button"
              disabled={!selectedReminderId}
              onClick={() => void setSelectedReminderStatus("sent")}
              type="button"
            >
              Mark Sent
            </button>
            <button
              className="ghost-button"
              disabled={!selectedReminderId}
              onClick={() => void deleteSelectedReminder()}
              type="button"
            >
              Delete
            </button>
            <button className="ghost-button" onClick={resetReminderDraft} type="button">
              Clear
            </button>
          </div>

          <div className="task-list-scroll">
            {isLoadingReminders && reminders.length === 0 ? <p className="muted-copy">Loading reminders...</p> : null}
            {!isLoadingReminders && reminders.length === 0 ? <p className="muted-copy">No reminders yet.</p> : null}

            {reminders.map((reminder) => (
              <button
                key={reminder.id}
                className={`task-row ${selectedReminderId === reminder.id ? "is-selected" : ""}`}
                onClick={() => selectReminder(reminder.id)}
                type="button"
              >
                <div className="task-row-main">
                  <h4>{reminder.title}</h4>
                  <p>{reminder.message || "No message"}</p>
                </div>
                <div className="task-row-meta">
                  <span className="resource-chip">{reminder.status}</span>
                  <span>{formatDateTime(reminder.remindAt)}</span>
                </div>
                {reminder.resourceId ? <div className="task-row-link">Resource: {reminder.resourceId}</div> : null}
              </button>
            ))}
          </div>
        </article>
      </div>
    </section>
  );
}
