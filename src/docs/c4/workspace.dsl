workspace "Payment Hub POC" "C4 model for ClickPay Hub POC" {
  model {
    person user "Client App" "Web/Mobile app that integrates with the Hub" {
      tags "External"
    }

    softwareSystem hub "Payment Hub" "POC that orchestrates payment requests and providers" {}

    // External dependencies
    softwareSystem sqs "Amazon SQS" "Message queue for orchestration" {
      tags "External","Queue"
    }
    softwareSystem pg "PostgreSQL" "Relational database" {
      tags "External","Database"
    }
    softwareSystem pix "PIX Provider (Mock)" "External payment provider" { tags "External" }
    softwareSystem card "Card Provider (Mock)" "External payment provider" { tags "External" }

    // Containers inside the Hub
    container ingress of hub "Ingress API" "Go (chi, validator, ulid)" "HTTP API for clients; creates PaymentIntent and outbox rows" {
      tags "Container"
    }
    container publisher of hub "Outbox Publisher" "Go (AWS SDK v2)" "Publishes outbox events to SQS" { tags "Container" }
    container worker of hub "Orchestrator Worker" "Go (AWS SDK v2, gobreaker, backoff)" "Consumes SQS, calls providers, updates DB" { tags "Container" }
    container webhook of hub "Webhook Receiver" "Go (chi)" "Receives provider callbacks and updates DB" { tags "Container" }
    container db of hub "PostgreSQL" "PostgreSQL 14+ (pgx)" "Stores payment_intent and outbox" { tags "Database" }

    // Relationships
    user -> ingress "Create payment / Query status" "HTTP/JSON"
    ingress -> db "Insert payment_intent + outbox" "SQL (pgx)"
    publisher -> db "Fetch unpublished outbox; mark published" "SQL"
    publisher -> sqs "Send messages" "AWS SDK v2"
    sqs -> worker "Receive messages" "AWS SDK v2"
    worker -> pix "Authorize/charge PIX" "HTTP (mock)"
    worker -> card "Authorize card" "HTTP (mock)"
    pix -> webhook "Callbacks /webhooks/pix" "HTTP"
    card -> webhook "Callbacks /webhooks/card" "HTTP"
    webhook -> db "Update status/provider fields" "SQL"
    ingress -> db "Get payment status" "SQL"
  }

  views {
    systemContext hub "System Context" {
      include *
      autolayout lr
      title "Payment Hub - System Context"
      description "Actors and external dependencies"
    }

    container hub "Container View" {
      include *
      autolayout lr
      title "Payment Hub - Containers"
      description "Containers and key integrations"
    }

    styles {
      element "External" { background "#ffffff"; color "#111111"; border "#666666" }
      element "Container" { background "#1168bd"; color "#ffffff" }
      element "Database" { shape "Cylinder"; background "#438dd5"; color "#ffffff" }
      element "Queue" { shape "Pipe"; background "#999999"; color "#ffffff" }
    }
  }
}

