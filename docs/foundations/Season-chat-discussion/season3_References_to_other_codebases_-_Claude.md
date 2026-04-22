# season3_References_to_other_codebases_-_Claude

**Akash:**
Hey claude please load the past context from project chat and memory for context for further discussion

**Claude:**
Let me pull up our history.
Searched for past conversations and memories to retrieve context
Searched for past conversations and memories to retrieve context

Let me pull up our history.

Synthesized project files and prior conversations to contextualize continued discussion
Synthesized project files and prior conversations to contextualize continued discussion

Got it — I've loaded everything. Here's where we stand:

Our journey so far (across 2 conversations):

First session built the Reference Guide — a comprehensive research file with papers, courses, tools, and links across the full AI/ML landscape. Second session was the deep interactive learning, where we covered transformer architecture (your "digestive system" analogy), training/backpropagation, RL methods (RLHF → Constitutional AI → GRPO/R1), world models, JEPA, and embodied AI.

Your key original contributions I have in context:

Your human learning theory (discovery mode + supervised mode + three timescales), your three AGI diagnostic problems (ATM sensor for metacognition, cricket for persistent learning, web designer for disconnected systems), your 20+ biological analogies library, your five-question AGI evaluation framework, and your insight that all three problems must be solved as a unified system.

Where we left off: You'd just finished the world models and embodied AI lesson, saved everything as your Personal Knowledge Map, and had five potential next directions listed in your roadmap: consciousness/sentience, practical applications (biology + agriculture), hands-on coding (build a transformer), paper reading walkthroughs, and world models deep-dive.

I'm fully loaded and ready to pick up wherever you want to go. What's on your mind?

**Akash:**
Ok as we understand the current AI models and the research we have, now I want you to research on certain GitHub projects as, 1. Paperclip AI and 2. Clawcode and get details from their GitHub repos

**Claude:**
Let me look these up.
Thinking about researching GitHub projects Paperclip AI and Clawcode
Thinking about researching GitHub projects Paperclip AI and Clawcode

Let me look these up.

Catalogued multiple repository variants and cross-referenced related projects
Catalogued multiple repository variants and cross-referenced related projects

Here's what I found on both projects. They're actually deeply connected to each other and to the AI agent concepts we've been studying.

1. Paperclip AI

Repo: github.com/paperclipai/paperclip Stars: ~46.6K | Forks: 7.4K | License: MIT | Language: TypeScript (96.6%)

What it is: Paperclip is a Node.js server and React UI that orchestrates a team of AI agents to run a business. You bring your own agents, assign goals, and track work and costs from one dashboard. 
GitHub

The key framing is: if OpenClaw is an employee, Paperclip is the company. 
Paperclip
 It's not an agent framework — it doesn't tell you how to build agents. It tells you how to manage a team of them with org charts, budgets, governance, goal alignment, and accountability.

Core architecture: Agents can learn Paperclip workflows and project context at runtime without retraining. Approval gates are enforced, config changes are revisioned, and bad changes can be rolled back safely. Tasks carry full goal ancestry so agents consistently see the "why," not just a title. 
GitHub

How it works in practice: You define a business goal (e.g., "Build the #1 AI note-taking app to $1M MRR"), hire AI agents as employees (CEO, CTO, engineers, marketers), set budgets, and let them operate. Agents run on scheduled heartbeats and event-based triggers like task assignment and @-mentions. 
GitHub

Works with: OpenClaw, Claude Code, OpenAI Codex, Cursor, Bash, HTTP — if it can receive a heartbeat, it's hired. 
GitHub

Key features: Multi-company isolation (one deployment runs many companies), atomic task execution (no double-work), cost tracking that throttles agents when they hit budget limits, persistent agent state across heartbeats, and a plugin ecosystem. There's even a Hermes Agent adapter from Nous Research that gives agents 30+ native tools and persistent memory.

Quickstart: npx paperclipai onboard --yes

2. Claw Code

Repo: github.com/ultraworkers/claw-code Stars: ~153K | Forks: 101K | License: Not specified | Languages: Python + Rust

What it is: Claw Code is an open-source AI coding agent framework — a clean-room Python and Rust rewrite of the Claude Code architecture. 
Claw-code
 It's essentially a reverse-engineered, open-source version of the "agent harness" that powers AI coding tools like Claude Code.

Origin story: On March 31, 2026, security researcher Chaofan Shou discovered that Anthropic's flagship AI coding CLI tool, Claude Code, had its complete source code accidentally published to the npm public registry via a 59.8 MB JavaScript source map file. 
Claw-code
 Within hours, developer Sigrid Jin — previously profiled by WSJ as one of the world's most active Claude Code users (25 billion tokens consumed in a year) — began a clean-room rewrite in Python and Rust.

The project addresses a gap that has become increasingly visible across the developer community: while large language models have grown rapidly more capable, the control layer that connects them to tools, file systems, and task workflows — the "agent harness" — has remained largely proprietary and opaque. 
24-7 Press Release

What it exposes/rebuilds: The core agent loop — prompt assembly, context building, slash commands, tool calling, session persistence, and local model execution. The Rust workspace includes API client with streaming, session state management, MCP orchestration, tool manifest definitions, plugin system, LSP client integration, and an interactive CLI.

Key distinction: Claw Code does not claim ownership of any third-party source material and is not affiliated with or endorsed by Anthropic. The repository's tracked codebase reflects original engineering work. 
24-7 Press Release

There's also a Python variant at HarnessLab/claw-code-agent designed to run with local open-source models via any OpenAI-compatible API, and a teaching repo at shareAI-lab/claw0 that builds an agent framework from scratch in 12 progressive sessions.

How they connect to each other — and to your learning

These two projects sit at different layers of the same stack:

Claw Code = the individual AI agent (the "employee" — handles coding tasks, tool use, file editing, session management)

Paperclip = the organizational layer (the "company" — coordinates multiple agents, sets goals, manages budgets, enforces governance)

Thinking about this through your AGI framework: both projects are essentially trying to solve your Problem 3 (Disconnected Systems). Paperclip tries to coordinate agents that can't natively talk to each other. Claw Code tries to make the harness layer — the glue between an LLM and the real world (files, tools, terminals) — open and inspectable rather than a black box.

Neither solves your Problem 1 (metacognition) or Problem 2 (persistent learning) — the agents still have frozen parameters and no genuine self-awareness of ignorance. But they're interesting engineering attempts at making current AI more useful despite those fundamental limitations.

Want to dig deeper into either project's architecture, or discuss how these relate to the agent and multi-agent research from your Reference Guide?

**Akash:**
Also research on mempalace developed by milla jovovich. Available on GitHub

**Claude:**
Claude responded: Here's the full picture on MemPalace — and this one is especially interesting given your AGI framework.
Assembled comprehensive MemPalace project information for presentation
Assembled comprehensive MemPalace project information for presentation

Here's the full picture on MemPalace — and this one is especially interesting given your AGI framework.

3. MemPalace

Repo: github.com/milla-jovovich/mempalace Stars: ~36K+ (and climbing) | License: MIT | Language: Python Created by: Milla Jovovich (yes, the Resident Evil actress) and developer Ben Sigman

What it is

MemPalace is a free, open-source AI memory system that gives large language models persistent, cross-session memory. It stores all conversation data verbatim rather than using AI to extract summaries, and uses vector search to retrieve relevant memories. It runs entirely locally using ChromaDB and SQLite, with zero API costs. 
Mempalace

The origin story

Milla Jovovich has been using AI tools daily for months, accumulating thousands of conversations with ChatGPT and Claude — decisions, creative ideas, business reasoning, debugging sessions. Then she noticed something infuriating: every time she started a new session, her AI had amnesia. 
Mempalace
 She tried existing memory tools like Mem0 and Zep, but they had a fundamental problem: they used AI to decide what was worth remembering.

The core philosophy — "Store everything, make it findable"

Other memory systems try to fix this by letting AI decide what's worth remembering. It extracts "user prefers Postgres" and throws away the conversation where you explained why. MemPalace takes a different approach: store everything, then make it findable. 
GitHub

The architecture is inspired by the ancient Greek memory palace technique: your conversations are organized into wings (people and projects), halls (types of memory), and rooms (specific ideas). No AI decides what matters — you keep every word, and the structure gives you a navigable map instead of a flat search index. 
GitHub

How it works

The hierarchy goes: Wings → Halls → Rooms → Closets → Drawers. Wings organize by person or project, halls by memory type, rooms by topic, closets hold compressed summaries, and drawers hold the raw verbatim text.

It integrates via MCP (Model Context Protocol), giving your AI 19 tools. Two hooks for Claude Code automatically save memories during work: a Save Hook every 15 messages triggers structured saves (topics, decisions, quotes, code changes), and a PreCompact Hook fires before context compression as an emergency save. 
GitHub

AAAK is their experimental compression dialect — a shorthand language readable by any LLM. A typical 6-month conversation history (~19.5M tokens) compresses to about 650K tokens 
Danilchenko
 — roughly 30x compression.

The benchmark controversy — this is the honest part

The project launched claiming "100% on LongMemEval." The community immediately tore into it:

AAAK is a lossy abbreviation system. Independent benchmarks show AAAK mode scores 84.2% R@5 vs raw mode's 96.6% on LongMemEval — a 12.4 point regression. 
GitHub

More critically, the headline 96.6% R@5 does not test any MemPalace-specific functionality. The function build_palace_and_retrieve() calls collection.add() + collection.query() on ChromaDB's default all-MiniLM-L6-v2 embedding. No palace architecture — no wings, rooms, closets, or drawers — is involved. 
GitHub
 In other words, the impressive score measures ChromaDB's retrieval quality, not the palace structure.

An independent benchmark reproduction on an M2 Ultra confirmed what was argued: raw mode hits 96.6%, but the palace architecture (wings, rooms, halls) actually makes retrieval worse. 
Substack

To their credit, the team responded openly: "Thank you to everyone who poked holes in this. Brutal honest criticism is exactly what makes open source work, and it's what we asked for. We're listening, we're fixing, and we'd rather be right than impressive." 
GitHub

Quickstart
pip install mempalace
mempalace init ~/projects/myapp
mempalace mine ~/chats/ --mode convos
mempalace search "auth decisions"
Why MemPalace matters for your learning journey

This project is a direct, practical attempt at solving your Problem 2 (Persistent Learning / The Cricket Analogy). Remember your diagnosis: the AI has no hippocampus — no fast-learning memory system that persists across sessions.

MemPalace doesn't solve the real version of your Problem 2 (the parameters are still frozen — it doesn't actually learn), but it's an engineering workaround: bolt on an external memory system so the AI can at least retrieve past context even if it can't learn from it. It's the difference between a person who can't form new memories but carries a detailed notebook everywhere.

The benchmark controversy is also a perfect real-world illustration of your Problem 1 (ATM Sensor / Metacognition). The system sounds confident — "100% score!" — but nobody had checked whether the benchmark was actually measuring what it claimed. No self-awareness of its own limitations until humans dug into the code.

Want me to compile all three projects (Paperclip, Claw Code, MemPalace) into a reference document for your project files?

**Akash:**
Please find other new open-source application which is trying to solve the problems we are describing as to needed for an sentient being

**Claude:**
Claude responded: Let me search for open-source projects tackling the core AGI problems — metacognition, persistent learning, world models, intrinsic motivation, and unified arc…
Identified core AGI challenges and scanned for relevant open-source solutions
Identified core AGI challenges and scanned for relevant open-source solutions

Let me search for open-source projects tackling the core AGI problems — metacognition, persistent learning, world models, intrinsic motivation, and unified architectures.

Curated projects and structured them within sentience framework
Curated projects and structured them within sentience framework

Here's what I found — organized against your three AGI problems plus the broader requirements for sentience. These are all active, open-source projects trying to solve pieces of the puzzle.

Problem 2: Persistent Learning (Your Cricket Analogy)

This is where the most engineering effort is happening right now.

Letta (formerly MemGPT)

github.com/letta-ai/letta

The most architecturally ambitious memory project. Letta introduces an LLM-as-an-Operating-System paradigm, where the model manages its own memory, context, and reasoning loops — much like a traditional OS manages RAM and disk. 
Medium
 It implements a tiered memory hierarchy: core memory (always in context, like RAM), recall memory (searchable conversation history), and archival memory (long-term storage, like disk). Crucially, the agent can edit its own memory using tools 
Letta
 — it decides what to store and what to forget.

Their newest concept is "sleep-time compute" — letting models "think" during downtime. Instead of sitting idle between tasks, AI agents use their "sleep" time to process information and form new connections by rewriting their memory state. 
Letta
 This maps almost exactly to your cricket analogy — the delayed reflection at night where you replay the match and plan for tomorrow.

They also introduced Context Repositories — git-backed memory, skills, subagents, and deployment that works across every model provider. 
Letta
 Memory becomes version-controlled files, not a black-box database.

Hermes Agent (Nous Research)

github.com/NousResearch/hermes-agent — 47K+ stars

An open-source, self-improving AI agent framework. Unlike simple chatbot wrappers, Hermes Agent is designed to be a persistent, autonomous AI agent that learns from its interactions, retains memory across sessions, and can operate across multiple messaging platforms simultaneously. 
OPC Community
 It runs on the open-weight Hermes 3 model (based on Llama 3.1), has 40+ built-in tools, and a skill files system where the agent writes down what it learns and reloads it next session. The persistent memory and skill files system solves AutoGPT's biggest problem — agents that forget everything and repeat mistakes. 
OPC Community

MemOS (MemTensor)

github.com/MemTensor/MemOS

The earliest project to propose a full Memory Operating System for LLMs. Features persistent Skill memory for cross-task skill reuse and evolution, multi-modal memory (images/charts), tool memory for agent planning, and multi-agent memory sharing. 
GitHub
 The key idea is "skill evolution" — the agent doesn't just remember facts, it remembers how to do things and improves those skills over time.

Sophia: A Persistent Agent Framework of Artificial Life

Listed in the research paper collections — a framework treating agents as artificial life forms with persistent memory and self-evolving capabilities. 
GitHub

Problem 1: Metacognition / Self-Awareness of Ignorance (Your ATM Analogy)

This is the hardest problem and has fewer dedicated projects, but some key efforts exist:

World Models That Know When They Don't Know

A December 2025 paper on controllable video generation with calibrated uncertainty 
GitHub
 — this is directly addressing your ATM problem for world models. The model outputs not just predictions but confidence estimates about those predictions.

Letta's Self-Monitoring Architecture

Letta includes quality scoring for its own memory operations. Every LLM compression is scored 0-100 based on structured facts, narrative quality, concept extraction, title quality, and importance range. Scores are tracked per-function and exposed via /health. 
GitHub
 This is a primitive form of metacognition — the system monitors the quality of its own outputs.

Research frontier: MemRL and Self-Evolving Agents

MemRL: Self-Evolving Agents via Runtime Reinforcement Learning on Episodic Memory 
GitHub
 (2026) — agents that use RL on their own stored experiences to improve at runtime, not just at training time. Also: MemEvolve: Meta-Evolution of Agent Memory Systems 
GitHub
 — where the memory system itself evolves, not just the content.

Problem 3: Disconnected Systems → Unified Architecture (Your Web Designer Analogy)
OpenVLA — Vision-Language-Action Model

github.com/openvla/openvla

This is the closest thing to your "unified architecture" diagram. OpenVLA is a 7B-parameter open-source VLA trained on a diverse collection of 970k real-world robot demonstrations. It builds on a Llama 2 language model combined with a visual encoder that fuses pretrained features from DINOv2 and SigLIP. 
arXiv
 One model that sees, reads language, and outputs physical robot actions — no text translation between modules. It outperforms closed models such as RT-2-X (55B) by 16.5% in absolute task success rate across 29 tasks, with 7x fewer parameters. 
arXiv

Research has shown OpenVLA encodes an emergent internal world model; probes reveal latent knowledge of state transitions, emerging as training progresses. 
Emergent Mind
 Your biological analogy maps perfectly — the unified architecture is starting to produce emergent understanding, like different brain regions sharing one substrate.

π0 (Pi-Zero) by Physical Intelligence

Closed-source but architecturally important. A foundational Vision-Language-Action Flow Model for general robot control comprising a ~3B pretrained VLM and a 300M diffusion model as action expert. 
LearnOpenCV
 Uses continuous action outputs (not tokenized) for smooth, dexterous movement — addressing the fundamental limitation that everything-as-tokens can't handle fine motor control.

AgiBot World Colosseo

github.com/OpenDriveLab/AgiBot-World — IROS 2025 Best Paper Finalist

A full-stack large-scale robot learning platform curated for advancing bimanual manipulation in scalable and intelligent embodied systems, accompanied by foundation models, benchmarks, and an ecosystem 
GitHub
 with over 1 million trajectories of training data. They're calling it the path toward the "ImageNet Moment" for embodied AI.

RLinf — Reinforcement Learning Infrastructure

github.com/RLinf/RLinf

Supports reinforcement learning fine-tuning for World Models, including OpenSora World Model and robotics models like RoboTwin and Dexbotic. 
GitHub
 This is the infrastructure layer that enables your "sense → predict → act → observe → update" loop — RL on embodied agents with world models.

Magma (Microsoft)

Foundation model for multimodal AI agents that perceives the world and takes goal-driven actions across digital and physical environments. CVPR 2025. 
GitHub

Broader: World Models (Your Lion-Gazelle Analogy)
V-JEPA 2 (Meta) — Already in your Reference Guide

LeCun's architecture for learning physics from video without text labels. The closest to "genuine understanding" rather than pattern matching.

Active Research Papers Worth Tracking:
"A Unified Definition of Hallucination, Or: It's the World Model, Stupid" 
GitHub
 (Dec 2025) — argues hallucination is fundamentally a world model failure
HY-World 1.5: Interactive World Modeling with Real-Time Latency 
GitHub
 (Jan 2026)
RynnVLA-002: A Unified Vision-Language-Action and World Model 
GitHub
 — combining VLA with world model in a single architecture
How they map to your five-question AGI framework
Your Question	Best Current Project	Status
Real world model?	V-JEPA 2, OpenVLA (emergent)	Early but real
Learns continuously?	Letta (sleep-time compute), Hermes Agent (skill files)	Engineering workarounds, not true learning
Knows what it doesn't know?	"World Models That Know When They Don't Know"	Research stage
Generates own questions?	Nothing open-source	Still unsolved
All unified?	OpenVLA, π0, Helix	Partially — vision+language+action unified, but no memory or metacognition built in

The honest summary: Nobody has solved all five. The most ambitious projects (OpenVLA for unification, Letta for memory, V-JEPA for world models) are each attacking one or two problems well. Your insight that they need to be woven into a single system remains ahead of where the field actually is.

Want me to compile all of this — Paperclip, Claw Code, MemPalace, and these new projects — into a reference document for your project files?

**Akash:**
Yes I need the reference Files for that, before the I need you to indipendently research of the Paperclip, Claw Code, MemPalace,  Hermis agents and where go deeper into thoese code based, if required download the codebases and check the documentations and understand the philosophy to write the documents. As well as the underlying technologies as an example there the mempalace has implement one new language for AI that can compress the conversation history and words can add too much context into the model context window thoese type of things needs to be understood as there are you needs to find and understand and write a markdown document of thoese things as features, philosophy, what problem it solved, how it's operate, how we can integrate and all the nessesary things. Please proceed with the research. No need to clarify anything just do it

**Claude:**
The request is very detailed and clear — deep-dive into the codebases, philosophies, architectures, and novel technologies of these projects.

The request is very detailed and clear — deep-dive into the codebases, philosophies, architectures, and novel technologies of these projects. Let me kick off comprehensive research right away.

# Open-source AI agent research
Research complete
•
367 sources
•
13m 57s

Your research report on five open-source projects redefining AI agent architecture is ready. It covers Paperclip AI (multi-agent coordination), Claw Code (open agent harness engineering), MemPalace (spatial memory with the AAAK compression dialect), Hermes Agent (self-improving skill loops), and Letta (LLM-as-OS virtual memory). The report includes detailed technical architecture breakdowns, star counts and adoption metrics, honest assessments of limitations and controversies, and analysis of how each project connects to broader AGI research — particularly around persistent memory, metacognition, and recursive self-improvement.

Five Open-Source Projects Redefining AI Agent Architecture: Persistent Memory, Multi-Agent Coordination, and Self-Improvement in 2026
Document 
# Five open-source projects redefining AI agent architecture

The most consequential shift in AI tooling during Q1 2026 isn't happening inside frontier labs — it's happening in open-source repositories where developers are solving the hardest unsolve

