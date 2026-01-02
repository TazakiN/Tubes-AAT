graph TD
    %% Header styling
    classDef header fill:#f9f,stroke:#333,stroke-width:2px,font-size:16px,font-weight:bold;
    Title[CITYCONNECT - RABBITMQ FLOW]:::header

    %% --- Actors & Triggers ---
    UserAction[👤 USER ACTIONS]

    %% --- Producer Section ---
    subgraph "REPORT-SERVICE CONTAINER (Producers - Go)"
        direction TB
        note1[Services perform: <br/>1. Save to DB<br/>2. Async Publish]

        RS_Create["ReportService.<br/>CreateReport()"]
        RS_Update["ReportService.<br/>UpdateStatus()"]
        VS_Vote["VoteService.<br/>CastVote()"]
    end

    UserAction -->|"Create Report"| RS_Create
    UserAction -->|"Update Status"| RS_Update
    UserAction -->|"Vote Report"| VS_Vote

    %% --- RabbitMQ Section ---
    subgraph "🐰 RABBITMQ BROKER"
        direction TB
        RMQ_Exch{{"Exchange:<br/>cityconnect.notifications<br/>(Topic Type)"}}

        Q_Create[("queue.report_created")]
        Q_Status[("queue.status_updates")]
        Q_Vote[("queue.vote_received")]

        %% Publishing paths with routing keys
        RS_Create --"rk: report.created"--> RMQ_Exch
        RS_Update --"rk: report.status.updated"--> RMQ_Exch
        VS_Vote --"rk: report.vote.received"--> RMQ_Exch

        %% Bindings to Queues
        RMQ_Exch -.->|Matches: report.created| Q_Create
        RMQ_Exch -.->|Matches: report.status.*| Q_Status
        RMQ_Exch -.->|Matches: report.vote.*| Q_Vote
    end

    %% --- Consumer Section ---
    subgraph "📥 NOTIFICATION CONSUMER (Background Worker - Go)"
        direction TB
        note2[Goroutine Workers]

        H_Create["handleReportCreated()<br/>(Action: Just ACK)"]
        H_Status["handleStatusUpdate()<br/>(Action: Parse, Save DB, Push SSE)"]
        H_Vote["handleVoteReceived()<br/>(Action: Parse, Save DB, Push SSE)"]

        %% Consuming messages
        Q_Create ==> H_Create
        Q_Status ==> H_Status
        Q_Vote ==> H_Vote
    end

    %% --- Broadcasting Section ---
    subgraph "📡 SSE HUB (Go)"
        SSE_Manager["SSE Hub Manager<br/>(Manages UserID -> Clients Map)"]
    end

    H_Status -->|"Push Notification"| SSE_Manager
    H_Vote -->|"Push Notification"| SSE_Manager

    %% --- Client Section ---
    subgraph "CLIENT SIDE (Browser)"
        subgraph "🖥️ FRONTEND (Next.js)"
            FE_Component["NotificationBell.tsx<br/>(EventSource Listener)"]
        end
        EndUser[👤 USER sees notification]
    end

    %% The long connection stream
    SSE_Manager == "SSE Stream (HTTP long-connection)<br/>GET /api/v1/notifications/stream" ===> FE_Component
    FE_Component -->|"Update UI (show badge)"| EndUser

    %% --- Summary Table as a separate subgraph for visual grouping ---
    subgraph "SUMMARY TABLE"
        direction LR
        T1["ReportService / VoteService<br/>(Role: PUBLISHER)"] --- T2["RabbitMQ<br/>(Role: BROKER)"]
        T2 --- T3["NotificationConsumer<br/>(Role: SUBSCRIBER)"]
        T3 --- T4["SSE Hub<br/>(Role: BROADCASTER)"]
        T4 --- T5["EventSource (Browser)<br/>(Role: LISTENER)"]
    end

    %% Links to summary (optional, for context)
    style note1 fill:#fff,stroke:none,font-style:italic
    style note2 fill:#fff,stroke:none,font-style:italic