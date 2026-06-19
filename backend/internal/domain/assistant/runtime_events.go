package assistant

const (
	RuntimeEventRunStarted         = "run_started"
	RuntimeEventContextBuilt       = "context_built"
	RuntimeEventMessageAppended    = "message_appended"
	RuntimeEventLLMInvoked         = "llm_invoked"
	RuntimeEventToolRequested      = "tool_requested"
	RuntimeEventToolBlocked        = "tool_blocked"
	RuntimeEventApprovalRequired   = "approval_required"
	RuntimeEventToolCompleted      = "tool_completed"
	RuntimeEventContextInvalidated = "context_invalidated"
	RuntimeEventProposalCreated    = "proposal_created"
	RuntimeEventMemoryUpdated      = "memory_updated"
	RuntimeEventRunCompleted       = "run_completed"
	RuntimeEventRunFailed          = "run_failed"

	RuntimeErrorContext                 = "context_error"
	RuntimeErrorProvider                = "provider_error"
	RuntimeErrorTool                    = "tool_error"
	RuntimeErrorGovernanceDenied        = "governance_denied"
	RuntimeErrorFinanceValidationFailed = "finance_validation_failed"
	RuntimeErrorApprovalRejected        = "approval_rejected"
	RuntimeErrorRuntime                 = "runtime_error"
)
