# Upgrade an Adopting Repository to a New Standard Version

Do not rerun the adopter with `--force` across an established repository.

## Process

1. Record the currently pinned standard commit.
2. Create a purpose-named work branch.
3. Compare the old and new standard releases.
4. Review changed controls, schemas, templates, and reusable workflows.
5. Merge applicable changes into the repository's project-specific files.
6. Update the standard commit in `REPOSITORY-ASSURANCE.json`.
7. Run policy, portable, fresh-clone, canonical, specialized, and historical
   validation as applicable.
8. Record deviations and newly applicable controls.
9. Merge through the normal pull-request process.
10. Formally accept the upgraded repository-assurance boundary when material.

A standard update must not silently rewrite historical project acceptance.
