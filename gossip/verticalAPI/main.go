package verticalapi

// type Id uint32

type RegisteredModule struct {
	mainToVert chan MainToVertNotification
}

type VertToMainRegister struct {
	data   GossipNotify
	module *RegisteredModule
}

type VertToMainAnnounce struct {
	data GossipAnnounce
}

type VertToMainValidation struct {
	data GossipValidation
}

type MainToVertNotification struct {
	data GossipNotification
}

func listen(
	addr string,
	vertToMainRegister chan VertToMainRegister,
	vertToMainAnnounce chan VertToMainAnnounce,
	vertToMainValidation chan VertToMainValidation,
) error {
	return nil
}
