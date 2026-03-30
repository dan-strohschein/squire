package estimate

// ComputeSize maps impact metrics to a T-shirt size.
// Thresholds are calibrated from benchmark data on SyndrDB (225K LOC)
// and Prometheus codebases.
func ComputeSize(r *EstimateResult) string {
	crossCutting := r.Locks + r.ErrorMaps

	// TINY: trivial change, 1-2 files, handful of functions, single package
	if r.Files <= 2 && r.Functions <= 5 && r.Packages <= 1 && crossCutting == 0 {
		return "TINY"
	}

	// SMALL: focused change, few files, limited blast radius
	if r.Files <= 5 && r.Functions <= 12 && r.Packages <= 2 && crossCutting <= 1 {
		return "SMALL"
	}

	// MEDIUM: multi-file change across a few packages
	if r.Files <= 12 && r.Functions <= 25 && r.Packages <= 4 && crossCutting <= 2 {
		return "MEDIUM"
	}

	// LARGE: significant change across many packages
	if r.Files <= 25 && r.Functions <= 50 && r.Packages <= 8 {
		return "LARGE"
	}

	// XLARGE: everything else
	return "XLARGE"
}
