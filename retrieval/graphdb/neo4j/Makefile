.PHONY: test integration-test clean start-neo4j stop-neo4j

# 运行单元测试
test:
	go test -v -race

# 运行集成测试
integration-test:
	@echo "Starting Neo4j..."
	@cd ../../../ && docker compose -f docker-compose.graphdb.yml up -d neo4j
	@echo "Waiting for Neo4j to be ready..."
	@sleep 15
	@echo "Running integration tests..."
	go test -v -tags=integration -timeout 5m
	@echo "Integration tests completed!"

# 运行所有测试
test-all: test integration-test

# 清理测试数据
clean:
	@echo "Cleaning up test data..."

# 启动 Neo4j
start-neo4j:
	@cd ../../../ && docker compose -f docker-compose.graphdb.yml up -d neo4j
	@echo "Neo4j starting..."
	@echo "Wait 15 seconds for Neo4j to be ready..."
	@sleep 15
	@echo "Neo4j is ready!"
	@echo "Browser: http://localhost:7474"
	@echo "Username: neo4j"
	@echo "Password: password123"

# 停止 Neo4j
stop-neo4j:
	@cd ../../../ && docker compose -f docker-compose.graphdb.yml stop neo4j

# 查看 Neo4j 日志
logs:
	@docker logs -f langchain-neo4j

# 检查 Neo4j 状态
status:
	@docker ps | grep neo4j || echo "Neo4j is not running"
