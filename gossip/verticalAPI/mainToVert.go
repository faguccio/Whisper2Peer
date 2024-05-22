package verticalapi

import vertTypes "gossip/verticalAPI/types"

// type Id uint32

// This struct serves as collection of data needed to handle / communicate with
// a registered module
type RegisteredModule struct {
	MainToVert chan<- MainToVertNotification
}

// Wrapper for the GossipNotify message used for the message passing vert > main
type VertToMainRegister struct {
	Data   vertTypes.GossipNotify
	Module *RegisteredModule
}

// Wrapper for the GossipAnnounce message used for the message passing vert > main
type VertToMainAnnounce struct {
	Data vertTypes.GossipAnnounce
}

// Wrapper for the GossipValidation message used for the message passing vert > main
type VertToMainValidation struct {
	Data vertTypes.GossipValidation
}

// Wrapper for the GossipNotification message used for the message passing vert < main
type MainToVertNotification struct {
	Data vertTypes.GossipNotification
}

// This struct is a collection of channels needed for the communication vert > main
type VertToMainChans struct {
	Register   chan VertToMainRegister
	Anounce    chan VertToMainAnnounce
	Validation chan VertToMainValidation
}
