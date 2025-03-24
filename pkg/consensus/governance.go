package consensus

import (
	"errors"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"
	
	"github.com/ConfirmixLabs/Confirmix-Labs/pkg/blockchain"
	"github.com/google/uuid"
)

// ProposalStatus represents the status of a governance proposal
type ProposalStatus string

const (
	ProposalStatusPending   ProposalStatus = "pending"   // Awaiting voting
	ProposalStatusApproved  ProposalStatus = "approved"  // Approved by vote
	ProposalStatusRejected  ProposalStatus = "rejected"  // Rejected by vote
	ProposalStatusExecuted  ProposalStatus = "executed"  // Successfully executed
	ProposalStatusFailed    ProposalStatus = "failed"    // Execution failed
	ProposalStatusCancelled ProposalStatus = "cancelled" // Cancelled by creator
)

// ProposalType represents the type of proposal
type ProposalType string

const (
	ProposalTypeAddValidator    ProposalType = "add_validator"    // Add validator
	ProposalTypeRemoveValidator ProposalType = "remove_validator" // Remove validator
	ProposalTypeChangeParameter ProposalType = "change_parameter" // Change system parameter
	ProposalTypeUpgradeSoftware ProposalType = "upgrade_software" // Protocol upgrade
	ProposalTypeTransferFunds   ProposalType = "transfer_funds"   // Transfer from treasury
)

// Vote represents a vote on a proposal
type Vote struct {
	Voter       string    // Address of the voter
	VotedAt     time.Time // When the vote was cast
	VotingPower *big.Int  // Voting power (based on token balance)
	InFavor     bool      // True if in favor, false if against
}

// Proposal represents a governance proposal
type Proposal struct {
	ID          string            // Unique ID
	Type        ProposalType      // Type of proposal
	Title       string            // Short title
	Description string            // Detailed description
	Creator     string            // Address of creator
	CreatedAt   time.Time         // Creation timestamp
	ExpiresAt   time.Time         // Expiration timestamp
	Status      ProposalStatus    // Current status
	Data        map[string]string // Associated data for execution
	Votes       map[string]*Vote  // Map of address to vote
	YesVotes    *big.Int          // Total voting power in favor
	NoVotes     *big.Int          // Total voting power against
	ExecutedAt  time.Time         // When it was executed (if applicable)
	Result      string            // Result message after execution
}

// GovernanceConfig represents governance system configuration
type GovernanceConfig struct {
	VotingPeriod     time.Duration // How long voting lasts
	ExecutionDelay   time.Duration // Delay before execution after approval
	QuorumPercentage uint64        // Required participation (0-100)
	ApprovalThreshold uint64       // Required approval percentage (0-100)
	MinProposalDeposit *big.Int    // Minimum tokens required to create proposal
}

// Governance represents the governance/DAO system
type Governance struct {
	blockchain        *blockchain.Blockchain
	validatorManager  *ValidatorManager
	proposals         map[string]*Proposal
	mutex             sync.RWMutex
	config            GovernanceConfig
	tokenSystem       TokenSystem // Interface for token operations
	defaultGovernance bool        // Whether governance is enabled by default
	adminOverride     bool        // Whether admins can override governance
}

// TokenSystem is an interface for token operations
type TokenSystem interface {
	GetBalance(address string) (*big.Int, error)
	TransferFrom(from, to string, amount *big.Int) error
	Lock(address string, amount *big.Int) error
	Unlock(address string, amount *big.Int) error
}

// NewGovernance creates a new governance system
func NewGovernance(bc *blockchain.Blockchain, vm *ValidatorManager, ts TokenSystem, config GovernanceConfig) *Governance {
	return &Governance{
		blockchain:        bc,
		validatorManager:  vm,
		proposals:         make(map[string]*Proposal),
		config:            config,
		tokenSystem:       ts,
		defaultGovernance: false, // Start with governance disabled
		adminOverride:     true,  // Start with admin override enabled
	}
}

// DefaultConfig returns the default governance configuration
func DefaultGovernanceConfig() GovernanceConfig {
	minDeposit := new(big.Int)
	minDeposit.SetString("1000000000000000000000", 10) // 1000 tokens
	
	return GovernanceConfig{
		VotingPeriod:      7 * 24 * time.Hour,  // 1 week
		ExecutionDelay:    24 * time.Hour,      // 1 day
		QuorumPercentage:  33,                  // 33% participation required
		ApprovalThreshold: 60,                  // 60% yes votes required
		MinProposalDeposit: minDeposit,
	}
}

// CreateProposal creates a new governance proposal
func (g *Governance) CreateProposal(creator string, proposalType ProposalType, title, description string, data map[string]string) (string, error) {
	// Check if governance is enabled
	if !g.defaultGovernance && !g.validatorManager.IsValidator(creator) {
		return "", errors.New("governance is not yet enabled for non-validators")
	}
	
	// Check minimum deposit requirement
	balance, err := g.tokenSystem.GetBalance(creator)
	if err != nil {
		return "", fmt.Errorf("could not check token balance: %v", err)
	}
	
	if balance.Cmp(g.config.MinProposalDeposit) < 0 {
		return "", fmt.Errorf("insufficient token balance for proposal (required: %s, have: %s)",
			g.config.MinProposalDeposit.String(), balance.String())
	}
	
	// Lock the deposit
	if err := g.tokenSystem.Lock(creator, g.config.MinProposalDeposit); err != nil {
		return "", fmt.Errorf("failed to lock proposal deposit: %v", err)
	}
	
	g.mutex.Lock()
	defer g.mutex.Unlock()
	
	// Create proposal ID
	proposalID := uuid.New().String()
	
	// Create new proposal
	proposal := &Proposal{
		ID:          proposalID,
		Type:        proposalType,
		Title:       title,
		Description: description,
		Creator:     creator,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(g.config.VotingPeriod),
		Status:      ProposalStatusPending,
		Data:        data,
		Votes:       make(map[string]*Vote),
		YesVotes:    big.NewInt(0),
		NoVotes:     big.NewInt(0),
	}
	
	g.proposals[proposalID] = proposal
	
	log.Printf("Proposal created: %s - %s (by %s)", proposalID, title, creator)
	return proposalID, nil
}

// CastVote casts a vote on a proposal
func (g *Governance) CastVote(proposalID string, voter string, inFavor bool) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	
	// Find the proposal
	proposal, exists := g.proposals[proposalID]
	if !exists {
		return errors.New("proposal not found")
	}
	
	// Check if proposal is still pending
	if proposal.Status != ProposalStatusPending {
		return fmt.Errorf("proposal is not pending (current status: %s)", proposal.Status)
	}
	
	// Check if voting period has expired
	if time.Now().After(proposal.ExpiresAt) {
		return errors.New("voting period has ended")
	}
	
	// Check if already voted
	if _, voted := proposal.Votes[voter]; voted {
		return errors.New("already voted on this proposal")
	}
	
	// Calculate voting power (token balance) - validators get 2x voting power
	votingPower, err := g.tokenSystem.GetBalance(voter)
	if err != nil {
		return fmt.Errorf("failed to get token balance: %v", err)
	}
	
	// Validator bonus
	if g.validatorManager.IsValidator(voter) {
		// Double the voting power for validators
		votingPower = new(big.Int).Mul(votingPower, big.NewInt(2))
	}
	
	// Create vote
	vote := &Vote{
		Voter:       voter,
		VotedAt:     time.Now(),
		VotingPower: votingPower,
		InFavor:     inFavor,
	}
	
	// Record the vote
	proposal.Votes[voter] = vote
	
	// Update totals
	if inFavor {
		proposal.YesVotes = new(big.Int).Add(proposal.YesVotes, votingPower)
	} else {
		proposal.NoVotes = new(big.Int).Add(proposal.NoVotes, votingPower)
	}
	
	log.Printf("Vote cast on proposal %s by %s: %v (power: %s)", 
		proposalID, voter, inFavor, votingPower.String())
	
	// Check if proposal can be finalized
	g.checkAndFinalizeProposal(proposal)
	
	return nil
}

// checkAndFinalizeProposal checks if a proposal should be finalized based on votes
func (g *Governance) checkAndFinalizeProposal(proposal *Proposal) {
	// Check if proposal is still pending
	if proposal.Status != ProposalStatusPending {
		return
	}
	
	// Calculate total votes
	totalVotes := new(big.Int).Add(proposal.YesVotes, proposal.NoVotes)
	
	// Get total token supply for quorum calculation
	totalSupply, err := g.getTotalTokenSupply()
	if err != nil {
		log.Printf("Error getting total token supply: %v", err)
		return
	}
	
	// Calculate quorum percentage
	quorumRatio := new(big.Int).Mul(totalVotes, big.NewInt(100))
	quorumRatio.Div(quorumRatio, totalSupply)
	
	// Check if quorum is reached
	if quorumRatio.Uint64() < g.config.QuorumPercentage {
		// Not enough participation yet
		return
	}
	
	// Calculate approval percentage
	approvalRatio := new(big.Int).Mul(proposal.YesVotes, big.NewInt(100))
	approvalRatio.Div(approvalRatio, totalVotes)
	
	// Determine result
	if approvalRatio.Uint64() >= g.config.ApprovalThreshold {
		proposal.Status = ProposalStatusApproved
		log.Printf("Proposal %s approved (%d%% in favor, %d%% participation)", 
			proposal.ID, approvalRatio.Uint64(), quorumRatio.Uint64())
		
		// Schedule execution after delay
		go g.scheduleProposalExecution(proposal.ID, g.config.ExecutionDelay)
	} else {
		proposal.Status = ProposalStatusRejected
		log.Printf("Proposal %s rejected (%d%% in favor, %d%% participation)", 
			proposal.ID, approvalRatio.Uint64(), quorumRatio.Uint64())
		
		// Return deposit to creator
		go g.returnProposalDeposit(proposal.Creator)
	}
}

// scheduleProposalExecution schedules a proposal for execution after a delay
func (g *Governance) scheduleProposalExecution(proposalID string, delay time.Duration) {
	time.Sleep(delay)
	
	g.mutex.Lock()
	proposal, exists := g.proposals[proposalID]
	g.mutex.Unlock()
	
	if !exists || proposal.Status != ProposalStatusApproved {
		return
	}
	
	err := g.executeProposal(proposal)
	
	g.mutex.Lock()
	defer g.mutex.Unlock()
	
	proposal.ExecutedAt = time.Now()
	
	if err != nil {
		proposal.Status = ProposalStatusFailed
		proposal.Result = fmt.Sprintf("Execution failed: %v", err)
		log.Printf("Proposal %s execution failed: %v", proposalID, err)
		
		// Return deposit to creator on failure
		go g.returnProposalDeposit(proposal.Creator)
	} else {
		proposal.Status = ProposalStatusExecuted
		proposal.Result = "Execution successful"
		log.Printf("Proposal %s executed successfully", proposalID)
		
		// Return deposit to creator on success
		go g.returnProposalDeposit(proposal.Creator)
	}
}

// executeProposal executes an approved proposal
func (g *Governance) executeProposal(proposal *Proposal) error {
	switch proposal.Type {
	case ProposalTypeAddValidator:
		// Add validator proposal
		address, exists := proposal.Data["address"]
		if !exists {
			return errors.New("validator address missing from proposal data")
		}
		
		humanProof, exists := proposal.Data["humanProof"]
		if !exists {
			return errors.New("human proof missing from proposal data")
		}
		
		return g.blockchain.RegisterValidator(address, humanProof)
		
	case ProposalTypeRemoveValidator:
		// Remove validator proposal
		address, exists := proposal.Data["address"]
		if !exists {
			return errors.New("validator address missing from proposal data")
		}
		
		return g.blockchain.RemoveValidator(address)
		
	case ProposalTypeChangeParameter:
		// Change parameter proposal
		// Implementation depends on what parameters are configurable
		return errors.New("parameter change proposals not yet implemented")
		
	case ProposalTypeUpgradeSoftware:
		// Software upgrade proposal
		// This would typically involve a coordinated upgrade process
		return errors.New("software upgrade proposals not yet implemented")
		
	case ProposalTypeTransferFunds:
		// Treasury transfer proposal
		to, exists := proposal.Data["to"]
		if !exists {
			return errors.New("recipient address missing from proposal data")
		}
		
		amountStr, exists := proposal.Data["amount"]
		if !exists {
			return errors.New("amount missing from proposal data")
		}
		
		amount := new(big.Int)
		if _, success := amount.SetString(amountStr, 10); !success {
			return errors.New("invalid amount format")
		}
		
		treasuryAddress := "confirmix_treasury" // Replace with actual treasury address
		return g.tokenSystem.TransferFrom(treasuryAddress, to, amount)
		
	default:
		return fmt.Errorf("unsupported proposal type: %s", proposal.Type)
	}
}

// returnProposalDeposit returns the deposit to the proposal creator
func (g *Governance) returnProposalDeposit(address string) {
	if err := g.tokenSystem.Unlock(address, g.config.MinProposalDeposit); err != nil {
		log.Printf("Error returning proposal deposit to %s: %v", address, err)
	} else {
		log.Printf("Proposal deposit returned to %s", address)
	}
}

// GetProposal returns a proposal by ID
func (g *Governance) GetProposal(proposalID string) (*Proposal, error) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	
	proposal, exists := g.proposals[proposalID]
	if !exists {
		return nil, errors.New("proposal not found")
	}
	
	return proposal, nil
}

// ListProposals returns all proposals with optional status filtering
func (g *Governance) ListProposals(statusFilter ...ProposalStatus) []*Proposal {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	
	var proposals []*Proposal
	
	if len(statusFilter) == 0 {
		// Return all proposals
		proposals = make([]*Proposal, 0, len(g.proposals))
		for _, proposal := range g.proposals {
			proposals = append(proposals, proposal)
		}
	} else {
		// Filter by status
		proposals = make([]*Proposal, 0)
		statusMap := make(map[ProposalStatus]bool)
		for _, status := range statusFilter {
			statusMap[status] = true
		}
		
		for _, proposal := range g.proposals {
			if statusMap[proposal.Status] {
				proposals = append(proposals, proposal)
			}
		}
	}
	
	return proposals
}

// SetDefaultGovernance enables/disables governance for all users
func (g *Governance) SetDefaultGovernance(enabled bool) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	g.defaultGovernance = enabled
	log.Printf("Default governance set to: %v", enabled)
}

// SetAdminOverride enables/disables admin override capability
func (g *Governance) SetAdminOverride(enabled bool) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	g.adminOverride = enabled
	log.Printf("Admin override set to: %v", enabled)
}

// UpdateConfig updates the governance configuration
func (g *Governance) UpdateConfig(config GovernanceConfig) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	g.config = config
	log.Printf("Governance configuration updated")
}

// getTotalTokenSupply gets the total token supply for quorum calculations
func (g *Governance) getTotalTokenSupply() (*big.Int, error) {
	// This is a placeholder - in a real implementation, you would query the
	// token contract or other mechanism to get the actual total supply
	totalSupply := new(big.Int)
	totalSupply.SetString("100000000000000000000000000", 10) // 100 million tokens
	return totalSupply, nil
} 