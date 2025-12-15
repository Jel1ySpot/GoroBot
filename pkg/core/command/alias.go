package command

func (r *Registry) CheckAlias(ctx *Context) {
	for _, alias := range r.Aliases {
		if alias.pattern.MatchString(ctx.String()) {
			target := ctx.Clone()
			if alias.transform != nil {
				target = alias.transform(target)
			}
			_ = r.Handler(target)
			return
		}
	}

	for i := range r.SubRegistries {
		r.SubRegistries[i].CheckAlias(ctx.Clone())
	}
}
