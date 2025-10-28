package main

// MailIngestPayload defines the standardized data structure expected by the backend service.
// This structure is designed to be populated by both the Graph API fetch response (after the webhook)
// and the VBA/PowerShell script (directly).
type MailIngestPayload struct {
    MessageID     string `json:"messageId"`
    SourceAccount string `json:"sourceAccount"`
    SenderEmail   string `json:"senderEmail"`
    Subject       string `json:"subject"`
    DataOrigin    string `json:"dataOrigin"` // "OutlookVBA" or "GraphWebhook"
}

// GraphNotification defines the structure of the *initial* notification 
// received directly from the Microsoft Graph webhook.
type GraphNotification struct {
    Value []struct {
        SubscriptionID string `json:"subscriptionId"`
        ClientState    string `json:"clientState"`
        Resource       string `json:"resource"`
        ResourceData struct {
            ID string `json:"id"` // This is the messageId we use for fetching
        } `json:"resourceData"`
    } `json:"value"`
}
