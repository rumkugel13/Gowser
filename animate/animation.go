package animate

import "fmt"

type Animation interface {
	Animate() string
}

type NumericAnimation struct {
	old_value        float64
	new_value        float64
	num_frames       int
	frame_count      int
	change_per_frame float64
}

func NewNumericAnimation(old_value, new_value float64, num_frames int) *NumericAnimation {
	return &NumericAnimation{
		old_value:        old_value,
		new_value:        new_value,
		num_frames:       num_frames,
		frame_count:      1,
		change_per_frame: (new_value - old_value) / float64(num_frames),
	}
}

func (n *NumericAnimation) Animate() string {
	n.frame_count++
	if n.frame_count >= n.num_frames {
		return ""
	}
	current_value := n.old_value + n.change_per_frame*float64(n.frame_count)
	return fmt.Sprintf("%f", current_value)
}
