# GitHub Release Ruleset Requirements

## Purpose

These controls protect development, accepted release branches, and ISRAS
acceptance identities from ordinary destructive changes.

## `dev` branch

The canonical `dev` branch must:

- block force pushes;
- block deletion;
- require pull-request integration for ordinary changes;
- require the applicable policy, portable, integration, and native operating
  system checks;
- restrict bypass to explicitly authorized release or recovery operations.

## `main` branch

The canonical `main` branch must:

- block force pushes;
- block deletion;
- represent the latest accepted release source boundary;
- move only through the controlled exact-commit promotion procedure;
- require verification that the intended target is the peeled target of the
  approved signed release tag.

## ISRAS acceptance tags

A tag ruleset must cover:

`isras-*`

The ruleset must:

- prevent ordinary update or deletion;
- restrict creation to authorized maintainers;
- require the release procedure to use an annotated signed tag;
- permit exceptional movement or deletion only through a separately recorded
  correction or recovery authorization.

## Verification

Release evidence must record the remote:

- tag object;
- peeled tag target;
- `main` commit;
- `dev` commit;
- signature result;
- applicable ruleset identity or administrative confirmation.
