package decision

// FindDependents returns all decisions that directly depend on the given ID.
func FindDependents(id string, all []Decision) []Decision {
	var result []Decision
	for _, d := range all {
		for _, dep := range d.DependsOn {
			if dep == id {
				result = append(result, d)
				break
			}
		}
	}
	return result
}

// FindTransitiveDependents returns all decisions reachable through dependency
// chains starting from the given ID (breadth-first). The source decision
// itself is not included.
func FindTransitiveDependents(id string, all []Decision) []Decision {
	visited := map[string]bool{id: true}
	queue := []string{id}
	var result []Decision

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		for _, d := range FindDependents(cur, all) {
			if !visited[d.ID] {
				visited[d.ID] = true
				result = append(result, d)
				queue = append(queue, d.ID)
			}
		}
	}
	return result
}

// FindDependencies returns all decisions that the given decision depends on.
func FindDependencies(d Decision, all []Decision) []Decision {
	lookup := make(map[string]Decision, len(all))
	for _, dec := range all {
		lookup[dec.ID] = dec
	}

	var result []Decision
	for _, depID := range d.DependsOn {
		if dep, ok := lookup[depID]; ok {
			result = append(result, dep)
		}
	}
	return result
}

// InvalidateDependent clears the answer on a decision and marks it as invalidated.
func InvalidateDependent(d *Decision) {
	d.Answer = nil
	d.Source = "invalidated"
	d.Delegated = false
}
