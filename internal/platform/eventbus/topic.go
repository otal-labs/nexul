package eventbus

// Topic is a published topic and the payload it carries; the event catalog generates the topic's schema from
// Payload's type (ADR 0137). A topic published with two payload types is declared once per type.
type Topic struct {
	Name string
	// Payload is a zero value of the payload type.
	Payload any
	// Description says when the event fires; it becomes the schema's description.
	Description string
}

// WireShaper is a payload type with its own MarshalJSON: WireShape returns a value of the type it encodes as.
type WireShaper interface {
	WireShape() any
}
