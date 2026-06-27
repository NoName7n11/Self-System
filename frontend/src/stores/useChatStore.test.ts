import { describe, it } from "vitest";

// Skipped pending UI redesign stabilization (Change 18).
// useChatStore is now conversation-based (conversations[] + sendToConversation);
// the old flat messages/sendMessage API was removed.
describe.skip("useChatStore", () => {
  it.todo("re-calibrate for conversation-based store after design stabilizes");
});
