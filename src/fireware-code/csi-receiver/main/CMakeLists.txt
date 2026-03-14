idf_component_register(SRCS "main.c"
                       PRIV_REQUIRES spi_flash
                       INCLUDE_DIRS "${IDF_PATH}/components/esp_driver_gpio/include"
)
