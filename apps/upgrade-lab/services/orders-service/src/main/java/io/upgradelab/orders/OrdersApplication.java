package io.upgradelab.orders;

import java.math.BigDecimal;
import java.util.List;
import java.util.Map;

import org.springframework.boot.CommandLineRunner;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.context.annotation.Bean;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@SpringBootApplication
public class OrdersApplication {
  public static void main(String[] args) {
    SpringApplication.run(OrdersApplication.class, args);
  }

  @Bean
  CommandLineRunner initialize(JdbcTemplate jdbcTemplate) {
    return args -> {
      Integer tableCount = jdbcTemplate.queryForObject(
          "select count(*) from information_schema.tables where table_name = 'orders'",
          Integer.class);
      if (tableCount != null && tableCount == 0) {
        jdbcTemplate.execute("""
            create table orders (
              id bigint identity(1,1) primary key,
              customer varchar(120) not null,
              sku varchar(64) not null,
              amount decimal(10, 2) not null
            )
            """);
      }
      Integer count = jdbcTemplate.queryForObject("select count(*) from orders", Integer.class);
      if (count != null && count == 0) {
        jdbcTemplate.update("insert into orders (customer, sku, amount) values (?, ?, ?)", "platform-team", "AKS-OPS-001", new BigDecimal("1200.00"));
        jdbcTemplate.update("insert into orders (customer, sku, amount) values (?, ?, ?)", "sre-team", "OBS-OTEL-002", new BigDecimal("450.00"));
      }
    };
  }
}

@RestController
class OrdersController {
  private final JdbcTemplate jdbcTemplate;

  OrdersController(JdbcTemplate jdbcTemplate) {
    this.jdbcTemplate = jdbcTemplate;
  }

  @GetMapping("/healthz")
  Map<String, String> healthz() {
    return Map.of("status", "ok", "service", "orders-service");
  }

  @GetMapping("/orders")
  List<Map<String, Object>> orders() {
    return jdbcTemplate.queryForList("select id, customer, sku, amount from orders order by id");
  }
}
