# Performance admission

## Current decision

Pending implementation measurements. No optimization and no speedup are
claimed. The intended mechanism performs one bounded-buffer pass over each
payload and hashes each payload byte once; archive memory must remain constant
with respect to total archive size.

With no baseline/candidate optimization pair, hotspot share `P`, hotspot
speedup `S_hotspot`, and an Amdahl prediction are not applicable. Inventing
them would be nonsense. Admission will report representative throughput,
allocations, and scaling without converting those measurements into a speedup
claim.
