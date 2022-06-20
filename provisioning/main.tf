terraform {
  required_providers {
    google = {
      source = "hashicorp/google"
    }
  }
}

provider "google" {
  credentials = file(var.credentials_file)
  version = "3.5.0"
  project = var.project
  ＃TF_VAR_project="<PROJECT_ID>"　の方がいいのかも
  region  = "asia-northeast1"
  zone    = "asia-northeast1-c"
}
resource "google_compute_network" "vpc_network" {
  name = "my-network"
}

resource "google_compute_instance" "vm_instance" {
  name         = "my-instance"
  machine_type = "n1-standard-1"
  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-9"
    }
  }
  network_interface {
    network = google_compute_network.vpc_network.self_link
    access_config {
      nat_ip = google_compute_address.vm_static_ip.address
    }
  }
}


resource "google_compute_address" "vm_static_ip" {
  name = "static-ip"
}

  
