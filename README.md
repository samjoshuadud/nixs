# nixs

nixs is a unified CLI tool for searching Nix packages, NixOS options, and Home Manager options from a single interface. It directly queries the search.nixos.org Elasticsearch backend to provide fast, formatted results without requiring local flake evaluation.

## Features

- Unified search for nixpkgs, NixOS options, and Home Manager options.
- Fast execution by querying the NixOS Elasticsearch backend directly.
- Relevant results using dis_max and cross_fields indexing.
- Clean pacman-style terminal output.
- Automatic schema probing to handle undocumented NixOS index version bumps.

## Installation

### Run without installation

You can run the tool directly using nix run:

```bash
nix run github:samjoshuadud/nixs -- firefox
```

### Install on NixOS (Flakes)

Add the repository to your flake inputs and include it in your system packages:

```nix
{
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    nixs.url = "github:samjoshuadud/nixs";
  };

  outputs = { self, nixpkgs, nixs, ... }: {
    nixosConfigurations.myhostname = nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      modules = [
        ./configuration.nix
        {
          environment.systemPackages = [
            nixs.packages.x86_64-linux.default
          ];
        }
      ];
    };
  };
}
```

### Install with Home Manager

Include it in your home.packages:

```nix
{ inputs, pkgs, ... }: {
  home.packages = [
    inputs.nixs.packages.${pkgs.system}.default
  ];
}
```

## Usage

Search packages (default):
```bash
nixs neovim
```

Search Home Manager options:
```bash
nixs --hm neovim
```

Search NixOS system options:
```bash
nixs --opt services.nginx
```

Show detailed package information with install commands:
```bash
nixs -i firefox
```

Search the stable channel (default is unstable):
```bash
nixs --stable firefox
```

Limit the number of results:
```bash
nixs -m 5 python
```
