// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package forge

import "slices"

// Registry looks up OAuth clients and reports which kinds are self-hosted.
type Registry interface {
	// Lookup returns the client registered for host.
	//
	// Parameters:
	//   - host: Git hostname, compared as given.
	//
	// Returns:
	//   - client: registered client, or the zero client when missing.
	//   - ok: true when host is registered.
	Lookup(host string) (Client, bool)

	// SelfHosted reports whether kind can be deployed off the vendor cloud.
	//
	// Parameters:
	//   - kind: forge kind to classify.
	//
	// Returns:
	//   - self-hosted: true only for GitLab, GitHub Enterprise, Gitea, and
	//     Forgejo.
	SelfHosted(kind Kind) bool
}

// Table is an immutable host-to-client registry.
//
// New and With return copies. Methods do not mutate the receiver map.
type Table struct {
	byHost map[string]Client
	hosts  []string
}

// Interface compliance for the registry used by Detect.
var _ Registry = (*Table)(nil)

// New returns the built-in public-host registry.
//
// Endpoint URLs are derived here, not in init. A malformed built-in URL
// returns an empty table instead of exiting the process.
//
// Returns:
//   - table: registry of every built-in host, including gist.github.com.
func New() *Table {
	clients, err := builtinClients()
	if err != nil {
		return &Table{
			byHost: map[string]Client{},
			hosts:  []string{},
		}
	}

	return tableFrom(clients)
}

// Hosts returns a stable sorted copy of the registered host names.
//
// Parameters:
//   - t: registry. A nil receiver returns nil.
//
// Returns:
//   - hosts: sorted host names. The slice is safe to mutate.
func (t *Table) Hosts() []string {
	if t == nil {
		return nil
	}

	return slices.Clone(t.hosts)
}

// Lookup returns a copy of the client registered for host.
//
// Parameters:
//   - host: Git hostname, compared as given.
//
// Returns:
//   - client: cloned client, or the zero client when missing.
//   - ok: true when host is registered.
func (t *Table) Lookup(host string) (Client, bool) {
	if t == nil || t.byHost == nil {
		return Client{}, false
	}

	client, ok := t.byHost[host]
	if !ok {
		return Client{}, false
	}

	return cloneClient(client), true
}

// SelfHosted reports whether kind can be deployed off the vendor cloud.
//
// Parameters:
//   - kind: forge kind to classify.
//
// Returns:
//   - self-hosted: true only for GitLab, GitHub Enterprise, Gitea, and Forgejo.
func (t *Table) SelfHosted(kind Kind) bool {
	return kind.IsSelfHosted()
}

// With returns a copy of the table with entries added or replaced.
//
// The receiver map is not mutated. Scope slices on entries are cloned.
//
// Parameters:
//   - entries: clients to add. Host is the map key.
//
// Returns:
//   - table: new registry containing the previous hosts plus entries.
func (t *Table) With(entries ...Client) *Table {
	next := t.clone()

	for _, entry := range entries {
		cloned := cloneClient(entry)
		if _, exists := next.byHost[cloned.Host]; !exists {
			next.hosts = append(next.hosts, cloned.Host)
		}

		next.byHost[cloned.Host] = cloned
	}

	slices.Sort(next.hosts)

	return next
}

// clone returns an independent copy of the table.
//
// Client scope slices are cloned. A nil table returns an empty table.
//
// Parameters:
//   - t: table to copy. Nil is treated as empty.
//
// Returns:
//   - *Table: copy that does not share the host map.
func (t *Table) clone() *Table {
	if t == nil || t.byHost == nil {
		return &Table{
			byHost: map[string]Client{},
			hosts:  []string{},
		}
	}

	next := &Table{
		byHost: make(map[string]Client, len(t.byHost)+1),
		hosts:  slices.Clone(t.hosts),
	}
	for host, client := range t.byHost {
		next.byHost[host] = cloneClient(client)
	}

	return next
}

// tableFrom indexes clients by host and sorts the host list.
//
// A repeated host keeps the first client.
//
// Parameters:
//   - clients: built-in or replacement clients.
//
// Returns:
//   - *Table: host-indexed table with a sorted host list.
func tableFrom(clients []Client) *Table {
	table := &Table{
		byHost: make(map[string]Client, len(clients)),
		hosts:  make([]string, 0, len(clients)),
	}
	for _, client := range clients {
		if _, exists := table.byHost[client.Host]; exists {
			continue
		}

		table.byHost[client.Host] = client
		table.hosts = append(table.hosts, client.Host)
	}

	slices.Sort(table.hosts)

	return table
}
