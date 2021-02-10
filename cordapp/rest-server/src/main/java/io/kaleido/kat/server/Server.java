package io.kaleido.kat.server;

import org.springframework.boot.Banner;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.WebApplicationType;
import org.springframework.boot.autoconfigure.EnableAutoConfiguration;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.context.ConfigurableApplicationContext;

/**
 * Our Spring Boot application.
 */
@SpringBootApplication
@EnableAutoConfiguration
public class Server {
    /**
     * Starts our Spring Boot application.
     */
    private static ConfigurableApplicationContext applicationContext ;
    public static void main(String[] args) {
        SpringApplication app = new SpringApplication(Server.class);
        app.setBannerMode(Banner.Mode.OFF);
        app.setWebApplicationType(WebApplicationType.SERVLET);
        applicationContext = app.run(args);
    }

    public static void stop() {
        SpringApplication.exit(applicationContext, () -> 0);
    }
}
