```mermaid
sequenceDiagram
    participant User
    participant Main
    participant Watcher
    participant PostgreSQL
    participant Stdout

    User->>Main: Execute with CLI flags
    Main->>Watcher: ParseFlags(build)
    Watcher-->>Main: FlagParam, ConnectionString
    
    Main->>Watcher: Run(ctx, fp, cp)
    
    Watcher->>PostgreSQL: Resolve DB list (if "all")
    PostgreSQL-->>Watcher: Database names
    
    alt master-only or replica-only
        Watcher->>PostgreSQL: Check role (pg_is_in_recovery)
        PostgreSQL-->>Watcher: Role status
    end
    
    par Parallel DB Processing (bounded by -j)
        Watcher->>PostgreSQL: Connect to DB1
        PostgreSQL-->>Watcher: Connection
        
        loop For each SQL query
            Watcher->>PostgreSQL: Execute query (with timeout)
            PostgreSQL-->>Watcher: Result rows
            
            loop For each row
                Watcher->>Watcher: Classify columns (labels vs metrics)
                Watcher->>Watcher: Format Prometheus output
                Watcher->>Stdout: Print metric line
            end
        end
        
        Watcher->>PostgreSQL: Close connection
    and
        Watcher->>PostgreSQL: Connect to DB2
        Note over Watcher,PostgreSQL: Same process for DB2...
    end
    
    Watcher-->>Main: Error or nil
    Main-->>User: Exit code (0 or 1)

    
