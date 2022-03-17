package com.example.app;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.SQLException;

@SpringBootApplication
@RestController
public class AppApplication {
  @RequestMapping("/")
  public String home() {
      return "Hello World!";
  }

  public static void main(String[] args) {
      String url = "jdbc:mysql://mysql:3306/mydatabase";
      String username = "root";
      String password = "mypassword";
      System.out.println("Connecting database...");

      try (Connection connection = DriverManager.getConnection(url, username, password)) {
          System.out.println("Database connected!");
      } catch (SQLException e) {
          throw new IllegalStateException("Cannot connect the database!", e);
      }

      SpringApplication.run(AppApplication.class, args);
  }
}


