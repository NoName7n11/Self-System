import { useEffect, useRef, useState } from "react";

import { useChatStore } from "../../stores/useChatStore";

export const presets = [
  "list resources",
  "list categories",
  "create category research | high-priority",
];

export async function submitChatMessage(
  rawInput: string,
  sendMessage: (content: string) => Promise<void>,
  onClear: () => void,
) {
  const value = rawInput.trim();
  if (value === "") {
    return;
  }

  await sendMessage(value);
  onClear();
}

export default function ChatDock() {
  const messages = useChatStore((state) => state.messages);
  const isSending = useChatStore((state) => state.isSending);
  const sendMessage = useChatStore((state) => state.sendMessage);

  const [input, setInput] = useState("");
  const logRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!logRef.current) {
      return;
    }

    logRef.current.scrollTo({
      top: logRef.current.scrollHeight,
      behavior: "smooth",
    });
  }, [messages]);

  const submit = async () => {
    await submitChatMessage(input, sendMessage, () => setInput(""));
  };

  return (
    <section className="chat-dock panel">
      <div className="panel-heading">
        <h2>Chat Layout</h2>
        <p>Command-oriented interaction bridge to backend workflows.</p>
      </div>

      <div className="preset-row">
        {presets.map((preset) => (
          <button key={preset} className="ghost-button" onClick={() => setInput(preset)} type="button">
            {preset}
          </button>
        ))}
      </div>

      <div className="chat-log" ref={logRef}>
        {messages.map((message) => (
          <article className={`chat-msg ${message.role === "assistant" ? "is-assistant" : "is-user"}`} key={message.id}>
            <header>{message.role === "assistant" ? "System" : "You"}</header>
            <p>{message.content}</p>
          </article>
        ))}
      </div>

      <div className="chat-input-row">
        <textarea
          onChange={(event) => setInput(event.target.value)}
          placeholder="Type a command..."
          rows={3}
          value={input}
        />
        <button className="primary-button" disabled={isSending} onClick={() => void submit()} type="button">
          {isSending ? "Running..." : "Send"}
        </button>
      </div>
    </section>
  );
}
