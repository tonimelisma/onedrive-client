Our intern attempted implementing parts of the @PRD.md with @ARCHITECTURE.md. Work is in @CHANGELOG.md 

Your job is to do a comprehensive code review and set things back on track. Look at both the detailed implementation as well as the big picture. Architecturally are we on the right track? Be tough but fair.

0. What criteria will you use to evaluate the code? What does a senior mobile engineer do in a code review?
1. Don't trust the markdown documents. Read the code as it is today on disk to understand its state and reason about it.
2. Explain all the files and the entire implementation and do the code review.
3. How would you do things differently? Provide both an overall architectural review, as well as more detailed implementation hot spots.
4. Create a detailed plan to get back on track, step by step on changes needed. If you were in charge of this, what would you do? Show it to me.
Thank you.
5. Where are the soft spots in your analysis? Enumerate them to me.
6. Read the rest of the files on disk required to harden the soft spots. Ensure you know each and every API that will be affected by your changes and have read the actual files on disk and know them before moving to the next step.
7. Implement the required changes.
8. Come up with a plan to test the changes. Implement the tests.
9. Build with verbose output and fix any build errors.
10. Run all the tests with verbose output and fix any test errors.
11. Update CHANGELOG.md, PRD.md, and ARCHITECTURE.md with any relevant changes. Document the detailed work you did, deviations from plan, code smells, shortcuts etc
12. Use your tools to do a git commit of everything changed. Ensure the comment is a one-liner. Then git push.

If you skip any step, you will be chastised. For instance, if you try to provide a plan before reading the code, a puppy dies.

In the beginning of your first message, repeat these numbered steps and then follow them to the letter.
