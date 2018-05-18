/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package cmd

import (
	"fmt"
	"os"
)

// Killer kills some resource and performs cleanup
type Killer func() error

// NewApplicationStopper invokes killer and stops application
func NewApplicationStopper(kill Killer) func() {
	return newStopper(kill, os.Exit)
}

type exitter func(code int)

func newStopper(kill Killer, exit exitter) func() {
	return func() {
		stop(kill, exit)
	}
}

func stop(kill Killer, exit exitter) {
	if err := kill(); err != nil {
		msg := fmt.Sprintf("Error while killing process: %v\n", err.Error())
		fmt.Fprintln(os.Stderr, msg)
		exit(1)
		return
	}

	fmt.Println("Good bye")
	exit(0)
}
