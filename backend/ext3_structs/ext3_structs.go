package ext3_structs

type Journal struct {
	J_count int32
	J_content [64]Information
}

type Information struct {
	I_operation [10]byte
	I_path [32]byte
	I_content [64]byte
	I_date float32
}