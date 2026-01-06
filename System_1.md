# <u> PERSONAL AI SYSTEM </u>

A personal AI assistant that keeps track of things important for my growth and productivity. Instead of relying on third-party APIs, I'll share content through my own custom messaging app.

## Data Sources

Content will be added by sharing links through my custom messaging app:
- Instagram posts/reels (via shared links)
- Web articles
- Any other URL-based content
- PDFs and images

[*Additional sources can be added as needed*]

## Features

### 1. Custom Command App
A dedicated mobile app for interacting with the system:
- Send links to save content
- Issue voice/text commands
- Quick actions (add task, query, search)
- Real-time responses via WebSocket
- Offline command queue

### 2. Infinite Canvas (Bubble Visualization)
An interactive visual space that automatically organizes saved content:
- **Auto-clustering**: AI groups similar items into "bubbles" (e.g., "Graphic Design", "AI Engineering")
- **Context connections**: Bubbles link when they share related concepts. Each bubble/node will be connencte 
- **Manual editing**: Ability to create, merge, or modify spaces
- **Context-aware AI**: Query specific bubbles for targeted responses
- **Visual navigation**: Navigate through interconnected knowledge spaces

### 3. Personal Assistant
AI-powered task and knowledge management:
- **Task management**: Natural language to-do lists with smart reminders
- **Memory**: Remembers previous thoughts, notes, and conversations
- **Proactive suggestions**: Context-aware recommendations
- **Learning**: Adapts to preferences and patterns over time
- **Behaves as +1**: Acts as an extension of me, not just a tool

[*Additional functionalities will be added iteratively*]

## Architecture

### Data Flow
```
Links Shared → Custom App → Content Extractor → Storage
                                                    ↓
                                            Vector Embeddings
                                                    ↓
                                    ┌───────────────┴───────────────┐
                                    ↓                               ↓
                            Infinite Canvas                    AI Query Engine
                         (Bubble Visualization)              (Online LLMs)
```

### LLM Strategy
- **Primary**: Online LLMs (Gemini, ChatGPT, Claude)
  - Cost-effective starting point
  - No GPU requirements
  - Fallback chain: Gemini → GPT-4o-mini → Claude
- **Future**: Local LLM option for sensitive data
  - Will implement when GPU resources available
  - Hybrid routing based on data sensitivity

### Tech Stack (Planned)

**Backend:**
- FastAPI (Python) or Node.js
- PostgreSQL (metadata) + Chroma/Pinecone (vectors)
- WebSocket for real-time communication

**Frontend:**
- Custom messaging app: React Native / Flutter
- Canvas interface: D3.js / Cytoscape.js / React Flow
- Web dashboard: React / Next.js

**AI/ML:**
- Content extraction: Trafilatura, Readability
- Embeddings: OpenAI, Google, or local models
- Clustering: K-means, DBSCAN for auto-grouping

## Implementation Phases

### Phase 1: MVP (Minimum Viable Product)
- [ ] Custom messaging app (basic UI)
- [ ] Link submission and storage
- [ ] Simple list view of saved content
- [ ] Basic task management
- [ ] Integration with one LLM (Gemini)

### Phase 2: Intelligence Layer
- [ ] Content extraction from shared links
- [ ] Vector embeddings and similarity search
- [ ] AI-powered suggestions
- [ ] Context-aware responses

### Phase 3: Canvas Visualization
- [ ] Infinite canvas implementation
- [ ] Auto-clustering algorithm
- [ ] Bubble creation and labeling
- [ ] Visual connections between bubbles
- [ ] Manual editing capabilities

### Phase 4: Advanced Features
- [ ] Multiple LLM provider support
- [ ] Advanced task scheduling
- [ ] Proactive notifications
- [ ] Learning from user behavior
- [ ] Local LLM integration (if resources allow)

## Key Challenges & Solutions

| Challenge | Solution |
|-----------|----------|
| Content extraction from links | Use Trafilatura/Readability + Instaloader for Instagram |
| LLM API costs | Smart caching, batch processing, cheaper models first |
| Canvas performance | Progressive rendering, virtualization for large datasets |
| Auto-grouping accuracy | Combine embeddings + LLM labeling for clusters |
| Real-time sync | WebSocket for commands, background jobs for processing |

## Success Metrics

- Time saved on organizing and finding information
- Number of useful AI suggestions acted upon
- Task completion rate improvement
- Subjective "feeling of being organized"

---

**Note**: This is a living document. Features and architecture will evolve based on learning and experimentation.
