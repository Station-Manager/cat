package cat

import (
	"github.com/Station-Manager/types"
)

func (s *Service) lineProcessor(shutdown <-chan struct{}) {
	for {
		select {
		case <-shutdown:
			return
		case state := <-s.processingChannel:
			s.LoggerService.DebugWith().Str("line", state.Data).Msg("Processing line")
			if len(state.Markers) == 0 {
				// No parsing required
				s.LoggerService.DebugWith().Str("line", state.Data).Msg("No parsing required")
				continue
			}

			status := types.CatStatus{}

			for _, marker := range state.Markers {
				start := marker.Index
				if start < 0 || start >= len(state.Data) {
					s.LoggerService.WarnWith().Int("index", start).Msg("marker index out of range; skipping marker")
					continue
				}

				end := marker.Index + marker.Length
				if end > len(state.Data) {
					s.LoggerService.WarnWith().Int("index", start).Int("length", marker.Length).Msg("marker end out of range; clamping to line end")
					end = len(state.Data)
				}
				if start >= end {
					s.LoggerService.DebugWith().Int("index", start).Int("length", marker.Length).Msg("empty slice for marker; skipping")
					continue
				}

				slice := state.Data[start:end]

				if len(marker.ValueMappings) == 0 {
					status[marker.Tag] = slice
				} else {
					mapped := ""
					for _, vm := range marker.ValueMappings {
						if slice == vm.Key {
							mapped = vm.Value
							break
						}
					}
					status[marker.Tag] = mapped // empty string if no mapping matched
				}
			}

			for {
				select {
				case <-shutdown:
					return
				case s.statusChannel <- status:
					continue
				default:
					if cap(s.statusChannel) == 0 {
						s.LoggerService.WarnWith().Msg("No consumer on unbuffered status channel, dropping status.")
						return
					}
					// Try to discard one oldest item non-blockingly, then loop to retry to re-send.
					select {
					case <-s.statusChannel:
						// discarded oldest; retry loop
					default:
						// nothing to discard (race); retry loop will attempt to send it again
					}
				}
			}
		}
	}
}
