package proto

func NewErrorResponse(err error) *MessageError {
	return &MessageError{Kind: KindError, Arg: ErrArg{err}}
}
