=== RUN ALL TESTS ===
go run . test

=== ONLY RUN SPECIFIC TEST SUITES ===
go run . test --test=deploy,init

=== ONLY RUN SPECIFIC SUB TESTS FOR A SPECIFIC TEST SUITE ===
for run . test --test-deploy=default,deploy
for run . test --test-init=use_chart,use_dockerfile
